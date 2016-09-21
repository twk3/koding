package stackplan

import (
	"koding/db/mongodb/modelhelper"
	"koding/kites/kloud/stack"

	"golang.org/x/net/context"
)

// Authenticate
func (bs *BaseStack) HandleAuthenticate(ctx context.Context) (interface{}, error) {
	var arg stack.AuthenticateRequest
	if err := bs.Req.Args.One().Unmarshal(&arg); err != nil {
		return nil, err
	}

	if err := arg.Valid(); err != nil {
		return nil, err
	}

	if err := bs.Builder.BuildCredentials(bs.Req.Method, bs.Req.Username, arg.GroupName, arg.Identifiers); err != nil {
		return nil, err
	}

	bs.Log.Debug("Fetched terraform data: koding=%+v, template=%+v", bs.Builder.Koding, bs.Builder.Template)

	resp := make(stack.AuthenticateResponse)

	for _, cred := range bs.Builder.Credentials {
		res := &stack.AuthenticateResult{}
		resp[cred.Identifier] = res

		if cred.Provider != bs.Planner.Provider {
			continue // ignore not ours credentials
		}

		if err := bs.Stack.VerifyCredential(cred.Credential); err != nil {
			res.Message = err.Error()
			continue
		}

		if err := modelhelper.SetCredentialVerified(cred.Identifier, true); err != nil {
			res.Message = err.Error()
			continue
		}

		res.Verified = true
	}

	bs.Log.Debug("Authenticate credentials result: %+v", resp)

	return resp, nil
}
