# Supervisor: A Process Control System

Backend workers can be started using
[supervisor](http://supervisord.org).

## Installation

You can install `supervisor` via `pip`.

```shell
pip install supervisor
```

## Starting up

`supervisord` by default looks up in current working directory for
`supervisord.conf` file first.  This file is generated by `configure`
script.  Make sure you executed `configure` script before running
`supervisord`.

`supervisord` starts programs defined in configuration immediately.

## Using `supervisorctl`

You can interact with `supervisor` via `supervisorctl` program.  It
reads configuration file in current working directory like
`supervisord`.

`supervisorctl` provides two interfaces, one is classic command line
options and the other is an interactive
[REPL](https://en.wikipedia.org/wiki/Read–eval–print_loop).

REPL mode can be entered by executing `supervisorctl` without any
commands.

```shell
supervisorctl
```

If `supervisord` is not running then following error output will be
printed:

```
error: <class 'xmlrpclib.Fault'>, <Fault 6: 'SHUTDOWN_STATE'>: file: /usr/lib/python2.7/xmlrpclib.py line: 794
```

By default, `supervisorctl` prints output of `status` command.

Example output during startup:

```
environment:janitor                   STARTING
environment:kloud                     STARTING
environment:kontrol                   STARTING
environment:terraformer               STARTING
environment:vmwatcher                 STARTING
socialapi:activityemail               STARTING
socialapi:algoliaconnector            STARTING
socialapi:collaboration               STARTING
socialapi:dailyemailnotifier          STARTING
socialapi:dispatcher                  STARTING
socialapi:eventsender                 STARTING
socialapi:gatekeeper                  STARTING
socialapi:integration                 STARTING
socialapi:mailsender                  STARTING
socialapi:notification                STARTING
socialapi:paymentwebhook              STARTING
socialapi:pinnedpost                  STARTING
socialapi:popularpost                 STARTING
socialapi:populartopic                STARTING
socialapi:privatemessageemailfeeder   STARTING
socialapi:privatemessageemailsender   STARTING
socialapi:realtime                    STARTING
socialapi:sitemapfeeder               STARTING
socialapi:sitemapgenerator            STARTING
socialapi                             STARTING
socialapi:team                        STARTING
socialapi:topicfeed                   STARTING
socialapi:topicmoderation             STARTING
socialapi:trollmode                   STARTING
socialapi:webhook                     STARTING
webserver:authworker                  STARTING
webserver:broker                      STARTING
webserver:gowebserver                 STARTING
webserver:rerouting                   STARTING
webserver:socialworker                STARTING
webserver:sourcemaps                  STARTING
webserver                             STARTING
```

Same command executed through command line options.

```
$ supervisorctl status
environment:janitor                   RUNNING   pid 3638, uptime 0:00:11
environment:kloud                     RUNNING   pid 3276, uptime 0:00:15
environment:kontrol                   RUNNING   pid 3271, uptime 0:00:15
environment:terraformer               RUNNING   pid 3274, uptime 0:00:15
environment:vmwatcher                 RUNNING   pid 3273, uptime 0:00:15
socialapi:activityemail               RUNNING   pid 3380, uptime 0:00:15
socialapi:algoliaconnector            RUNNING   pid 3327, uptime 0:00:15
socialapi:collaboration               RUNNING   pid 3382, uptime 0:00:15
socialapi:dailyemailnotifier          RUNNING   pid 3416, uptime 0:00:15
socialapi:dispatcher                  RUNNING   pid 3391, uptime 0:00:15
socialapi:eventsender                 RUNNING   pid 3362, uptime 0:00:15
socialapi:gatekeeper                  RUNNING   pid 3277, uptime 0:00:15
socialapi:integration                 RUNNING   pid 3326, uptime 0:00:15
socialapi:mailsender                  RUNNING   pid 3292, uptime 0:00:15
socialapi:notification                RUNNING   pid 3314, uptime 0:00:15
socialapi:paymentwebhook              RUNNING   pid 3384, uptime 0:00:15
socialapi:pinnedpost                  RUNNING   pid 3281, uptime 0:00:15
socialapi:popularpost                 RUNNING   pid 3361, uptime 0:00:15
socialapi:populartopic                RUNNING   pid 3392, uptime 0:00:15
socialapi:privatemessageemailfeeder   RUNNING   pid 3290, uptime 0:00:15
socialapi:privatemessageemailsender   RUNNING   pid 3287, uptime 0:00:15
socialapi:realtime                    RUNNING   pid 3312, uptime 0:00:15
socialapi:sitemapfeeder               RUNNING   pid 3360, uptime 0:00:15
socialapi:sitemapgenerator            RUNNING   pid 3342, uptime 0:00:15
socialapi                             RUNNING   pid 3280, uptime 0:00:15
socialapi:team                        RUNNING   pid 3417, uptime 0:00:15
socialapi:topicfeed                   RUNNING   pid 3289, uptime 0:00:15
socialapi:topicmoderation             RUNNING   pid 3279, uptime 0:00:15
socialapi:trollmode                   RUNNING   pid 3359, uptime 0:00:15
socialapi:webhook                     RUNNING   pid 3383, uptime 0:00:15
webserver:authworker                  RUNNING   pid 3446, uptime 0:00:15
webserver:broker                      RUNNING   pid 3448, uptime 0:00:15
webserver:gowebserver                 RUNNING   pid 3476, uptime 0:00:15
webserver:rerouting                   RUNNING   pid 3418, uptime 0:00:15
webserver:socialworker                RUNNING   pid 3482, uptime 0:00:15
webserver:sourcemaps                  RUNNING   pid 3419, uptime 0:00:15
webserver                             RUNNING   pid 3449, uptime 0:00:15
```

Commands listed below are useful during development

- start [all|program-name]
- stop [all|program-name]
- restart [all|program-name]
- reread
- reload
- shutdown
- exit

`start`, `stop`, and `restart` commands work on programs run by
`supervisord`.

You can execute `supervisorctl restart all` after pulling changes to
update workers running on your development environment.

`reread` command reads configuration file and updates supervisord
accordingly.  `reload` command, additional to `reread`, restarts all
programs which is useful when `supervisor` configuration is changed.

`shutdown` command kills `supervisord` process which is parent of
programs defined in configuration.  You do not need to kill supervisor
daemon during development cycle.  Stopping workers via `stop all`
command is probably serve you well for most of the cases.

`exit` command exits from interactive mode of `supervisorctl`.

## Logs

You can tail logs via following command after running `supervisord`.

```shell
tail -fq .logs/*
```
