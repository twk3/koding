package kloud

import (
	"errors"
	_ "expvar"
	"fmt"
	"io/ioutil"
	"log"
	_ "net/http/pprof"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/context"

	"koding/artifact"
	"koding/db/mongodb/modelhelper"
	"koding/httputil"
	"koding/kites/common"
	"koding/kites/keygen"
	"koding/kites/kloud/api/amazon"
	"koding/kites/kloud/contexthelper/publickeys"
	"koding/kites/kloud/contexthelper/session"
	"koding/kites/kloud/dnsstorage"
	"koding/kites/kloud/keycreator"
	"koding/kites/kloud/pkg/dnsclient"
	"koding/kites/kloud/provider"
	awsprovider "koding/kites/kloud/provider/aws"
	"koding/kites/kloud/queue"
	"koding/kites/kloud/stack"
	"koding/kites/kloud/stackplan"
	"koding/kites/kloud/stackplan/stackcred"
	"koding/kites/kloud/terraformer"
	"koding/kites/kloud/userdata"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/koding/kite"
	kiteconfig "github.com/koding/kite/config"
	"github.com/koding/logging"
)

//go:generate go run genimport.go -o import.go
//go:generate go fmt import.go

// Name holds kite name
var Name = "kloud"

// Kloud represents a configured kloud kite.
type Kloud struct {
	Kite   *kite.Kite
	Stack  *stack.Kloud
	Keygen *keygen.Server
}

// Config defines the configuration that Kloud needs to operate.
type Config struct {
	// ---  KLOUD SPECIFIC ---
	IP          string
	Port        int
	Region      string
	Environment string

	// Connect to Koding mongodb
	MongoURL string `required:"true"`

	// CredentialEndpoint is an API for managing stack credentials.
	CredentialEndpoint string

	// --- DEVELOPMENT CONFIG ---
	// Show version and exit if enabled
	Version bool

	// Enable debug log mode
	DebugMode bool

	// Enable production mode, operates on production channel
	ProdMode bool

	// Enable test mode, disabled some authentication checks
	TestMode bool

	// Defines the base domain for domain creation
	HostedZone string `required:"true"`

	// MaxResults limits the max items fetched per page for each
	// AWS Describe* API calls.
	MaxResults int `default:"500"`

	// --- KLIENT DEVELOPMENT ---
	// KontrolURL to connect and to de deployed with klient
	KontrolURL string `required:"true"`

	// KlientURL overwrites the Klient deb url returned by userdata.GetLatestDeb
	// method.
	KlientURL string

	// TunnelURL overwrites default tunnelserver url. Used by vagrant provider.
	TunnelURL string

	// Private key to create kite.key
	PrivateKey string `required:"true"`

	// Public key to create kite.key
	PublicKey string `required:"true"`

	// Private and public key to put a ssh key into the users VM's so we can
	// have access to it. Note that these are different then from the Kontrol
	// keys.
	UserPublicKey  string `required:"true"`
	UserPrivateKey string `required:"true"`

	// Keygen configuration.
	KeygenAccessKey string
	KeygenSecretKey string
	KeygenBucket    string
	KeygenRegion    string        `default:"us-east-1"`
	KeygenTokenTTL  time.Duration `default:"3h"`

	// --- KONTROL CONFIGURATION ---
	Public      bool   // Try to register with a public ip
	RegisterURL string // Explicitly register with this given url

	AWSAccessKeyId     string
	AWSSecretAccessKey string

	SLUsername string
	SLAPIKey   string

	JanitorSecretKey     string
	VmwatcherSecretKey   string
	KloudSecretKey       string
	TerraformerSecretKey string
}

// New gives new, registered kloud kite.
//
// If conf contains invalid or missing configuration, it return non-nil error.
func New(conf *Config) (*Kloud, error) {
	k := kite.New(stack.NAME, stack.VERSION)
	k.Config = kiteconfig.MustGet()
	k.Config.Port = conf.Port

	k.ClientFunc = httputil.ClientFunc(conf.DebugMode)

	if conf.DebugMode {
		k.SetLogLevel(kite.DEBUG)
	}

	if conf.Region != "" {
		k.Config.Region = conf.Region
	}

	if conf.Environment != "" {
		k.Config.Environment = conf.Environment
	}

	// TODO(rjeczalik): refactor modelhelper methods to not use global DB
	modelhelper.Initialize(conf.MongoURL)

	sess, err := newSession(conf, k)
	if err != nil {
		return nil, err
	}

	authUsers := map[string]string{
		"kloudctl":  conf.KloudSecretKey,
		"janitor":   conf.JanitorSecretKey,
		"vmwatcher": conf.VmwatcherSecretKey,
	}

	var credURL *url.URL

	if conf.CredentialEndpoint != "" {
		if u, err := url.Parse(conf.CredentialEndpoint); err == nil {
			credURL = u
		}
	}

	if credURL == nil {
		sess.Log.Warning(`disabling "Sneaker" for storing stack credential data`)
	}

	storeOpts := &stackcred.StoreOptions{
		MongoDB: sess.DB,
		Log:     sess.Log.New("stackcred"),
		CredURL: credURL,
		Client:  httputil.DefaultRestClient(conf.DebugMode),
	}

	bp := &provider.BaseProvider{
		DB:             sess.DB,
		Log:            sess.Log,
		Kite:           sess.Kite,
		Userdata:       sess.Userdata,
		Debug:          conf.DebugMode,
		KloudSecretKey: conf.KloudSecretKey,
		CredStore:      stackcred.NewStore(storeOpts),
		TunnelURL:      conf.TunnelURL,
	}

	// TODO(rjeczalik): refactor queue to work for any provider
	awsProvider := &awsprovider.Provider{
		BaseProvider: bp.New("aws"),
	}

	go runQueue(awsProvider, sess, conf)

	stats := common.MustInitMetrics(Name)

	kld := stack.New()
	kld.ContextCreator = func(ctx context.Context) context.Context {
		return session.NewContext(ctx, sess)
	}
	kld.Metrics = stats

	userPrivateKey, userPublicKey := userMachinesKeys(conf.UserPublicKey, conf.UserPrivateKey)

	// RSA key pair that we add to the newly created machine for
	// provisioning.
	kld.PublicKeys = &publickeys.Keys{
		KeyName:    publickeys.DeployKeyName,
		PrivateKey: userPrivateKey,
		PublicKey:  userPublicKey,
	}
	kld.DomainStorage = sess.DNSStorage
	kld.Domainer = sess.DNSClient
	kld.Locker = bp
	kld.Log = sess.Log
	kld.SecretKey = conf.KloudSecretKey

	for name, fn := range provider.All {
		p := fn(bp.New(name))

		err = kld.AddProvider(name, p)
		if err != nil {
			return nil, err
		}

		stackplan.MetaFuncs[name] = p.Cred
	}

	var gwSrv *keygen.Server
	if conf.KeygenAccessKey != "" && conf.KeygenSecretKey != "" {
		cfg := &keygen.Config{
			AccessKey:  conf.KeygenAccessKey,
			SecretKey:  conf.KeygenSecretKey,
			Region:     conf.KeygenRegion,
			Bucket:     conf.KeygenBucket,
			AuthExpire: conf.KeygenTokenTTL,
			AuthFunc:   kld.ValidateUser,
			Kite:       k,
		}

		gwSrv = keygen.NewServer(cfg)
	} else {
		k.Log.Warning(`disabling "keygen" methods due to missing S3/STS credentials`)
	}

	// Teams/stack handling methods
	k.HandleFunc("plan", kld.Plan)
	k.HandleFunc("apply", kld.Apply)
	k.HandleFunc("migrate", kld.Migrate)
	k.HandleFunc("describeStack", kld.Status)
	k.HandleFunc("authenticate", kld.Authenticate)
	k.HandleFunc("bootstrap", kld.Bootstrap)

	// Single machine handling
	k.HandleFunc("build", kld.Build)
	k.HandleFunc("destroy", kld.Destroy)
	k.HandleFunc("stop", kld.Stop)
	k.HandleFunc("start", kld.Start)
	k.HandleFunc("reinit", kld.Reinit)
	k.HandleFunc("restart", kld.Restart)
	k.HandleFunc("info", kld.Info)
	k.HandleFunc("event", kld.Event)
	k.HandleFunc("resize", kld.Resize)

	// Snapshot functionality
	k.HandleFunc("createSnapshot", kld.CreateSnapshot)
	k.HandleFunc("deleteSnapshot", kld.DeleteSnapshot)

	// Domain records handling methods
	k.HandleFunc("domain.set", kld.DomainSet)
	k.HandleFunc("domain.unset", kld.DomainUnset)
	k.HandleFunc("domain.add", kld.DomainAdd)
	k.HandleFunc("domain.remove", kld.DomainRemove)

	// Klient proxy methods
	k.HandleFunc("admin.add", kld.AdminAdd)
	k.HandleFunc("admin.remove", kld.AdminRemove)

	k.HandleHTTPFunc("/healthCheck", artifact.HealthCheckHandler(Name))
	k.HandleHTTPFunc("/version", artifact.VersionHandler())

	for worker, key := range authUsers {
		worker, key := worker, key
		k.Authenticators[worker] = func(r *kite.Request) error {
			if r.Auth.Key != key {
				return errors.New("wrong secret key passed, you are not authenticated")
			}
			return nil
		}
	}

	if conf.DebugMode {
		// This should be actually debug level 2. It outputs every single Kite
		// message and enables the kite debugging system. So enable it only if
		// you need it.
		// k.SetLogLevel(kite.DEBUG)
		k.Log.Info("Debug mode enabled")
	}

	if conf.TestMode {
		k.Log.Info("Test mode enabled")
	}

	registerURL := k.RegisterURL(!conf.Public)
	if conf.RegisterURL != "" {
		u, err := url.Parse(conf.RegisterURL)
		if err != nil {
			return nil, fmt.Errorf("Couldn't parse register url: %s", err)
		}

		registerURL = u
	}

	if err := k.RegisterForever(registerURL); err != nil {
		return nil, err
	}

	return &Kloud{
		Kite:   k,
		Stack:  kld,
		Keygen: gwSrv,
	}, nil
}

func newSession(conf *Config, k *kite.Kite) (*session.Session, error) {
	c := credentials.NewStaticCredentials(conf.AWSAccessKeyId, conf.AWSSecretAccessKey, "")

	kontrolPrivateKey, kontrolPublicKey := kontrolKeys(conf)

	klientFolder := "development/latest"
	if conf.ProdMode {
		k.Log.Info("Prod mode enabled")
		klientFolder = "production/latest"
	}

	k.Log.Info("Klient distribution channel is: %s", klientFolder)

	// Credential belongs to the `koding-kloud` user in AWS IAM's
	sess := &session.Session{
		DB:   modelhelper.Mongo,
		Kite: k,
		Userdata: &userdata.Userdata{
			Keycreator: &keycreator.Key{
				KontrolURL:        getKontrolURL(conf.KontrolURL),
				KontrolPrivateKey: kontrolPrivateKey,
				KontrolPublicKey:  kontrolPublicKey,
			},
			KlientURL: conf.KlientURL,
			Bucket:    userdata.NewBucket("koding-klient", klientFolder, c),
		},
		Terraformer: &terraformer.Options{
			Endpoint:  "http://127.0.0.1:2300/kite",
			SecretKey: conf.TerraformerSecretKey,
			Kite:      k,
		},
		Log: logging.NewCustom("kloud", conf.DebugMode),
	}

	sess.DNSStorage = dnsstorage.NewMongodbStorage(sess.DB)

	if conf.AWSAccessKeyId != "" && conf.AWSSecretAccessKey != "" {

		dnsOpts := &dnsclient.Options{
			Creds:      c,
			HostedZone: conf.HostedZone,
			Log:        logging.NewCustom("kloud-dns", conf.DebugMode),
			Debug:      conf.DebugMode,
		}

		dns, err := dnsclient.NewRoute53Client(dnsOpts)
		if err != nil {
			return nil, err
		}

		sess.DNSClient = dns

		opts := &amazon.ClientOptions{
			Credentials: c,
			Regions:     amazon.ProductionRegions,
			Log:         logging.NewCustom("kloud-koding", conf.DebugMode),
			MaxResults:  int64(conf.MaxResults),
			Debug:       conf.DebugMode,
		}

		ec2clients, err := amazon.NewClients(opts)
		if err != nil {
			return nil, err
		}

		sess.AWSClients = ec2clients
	}

	return sess, nil
}

func runQueue(aws stack.Provider, sess *session.Session, conf *Config) {
	q := &queue.Queue{
		Log: sess.Log.New("queue"),
	}

	if p, ok := aws.(*awsprovider.Provider); ok {
		q.AwsProvider = p
	}

	// TODO(rjeczalik): move to config
	interv := 5 * time.Second
	if conf.ProdMode {
		interv = time.Second / 2
	}

	go q.RunCheckers(interv)
}

func userMachinesKeys(publicPath, privatePath string) (string, string) {
	pubKey, err := ioutil.ReadFile(publicPath)
	if err != nil {
		log.Fatalln(err)
	}
	publicKey := string(pubKey)

	privKey, err := ioutil.ReadFile(privatePath)
	if err != nil {
		log.Fatalln(err)
	}
	privateKey := string(privKey)

	return strings.TrimSpace(privateKey), strings.TrimSpace(publicKey)
}

func kontrolKeys(conf *Config) (string, string) {
	pubKey, err := ioutil.ReadFile(conf.PublicKey)
	if err != nil {
		log.Fatalln(err)
	}
	publicKey := string(pubKey)

	privKey, err := ioutil.ReadFile(conf.PrivateKey)
	if err != nil {
		log.Fatalln(err)
	}
	privateKey := string(privKey)

	return privateKey, publicKey
}

func getKontrolURL(ownURL string) string {
	// read kontrolURL from kite.key if it doesn't exist.
	kontrolURL := kiteconfig.MustGet().KontrolURL

	if ownURL != "" {
		u, err := url.Parse(ownURL)
		if err != nil {
			log.Fatalln(err)
		}

		kontrolURL = u.String()
	}

	return kontrolURL
}
