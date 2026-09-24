# analytics-pipeline

Generate swagger docs:

    swag init -g api.go -o docs -dir pkg/api --parseDependency --ot json

## MongoDB configuration

| Env | Default | |
|---|---|---|
| `MONGO_URL` | `mongodb://localhost:27017` | full connection string, passed to the driver unchanged; must not contain credentials |
| `MONGO_USER` | empty | no authentication when empty |
| `MONGO_PASSWORD` | empty | required when `MONGO_USER` is set; masked in the logged config |
| `MONGO_AUTH_SOURCE` | `admin` | when `MONGO_USER` is set, replaces `authSource` and `authMechanism` from `MONGO_URL` (mechanism is negotiated) |
| `MONGO_DATABASE` | `analytics_pipeline` | must not be empty; collection is `pipelines` |

In a config file these are `mongo.url`, `mongo.user`, `mongo.password`, `mongo.auth_source` and `mongo.database`.

The config is logged as JSON at startup, so credentials belong in `MONGO_USER`/`MONGO_PASSWORD`, never in `MONGO_URL`. Startup fails unless an authenticated `listCollections` on `MONGO_DATABASE` succeeds within 10 seconds.

`go test -short ./...` skips the auth test. To run it, point it at a server with access control, a readWrite user on `analytics_pipeline` and a readWrite user on another database, both created in `admin`:

    MONGO_AUTH_TEST_URL=mongodb://127.0.0.1:27017/?directConnection=true \
    MONGO_AUTH_TEST_USER=... MONGO_AUTH_TEST_PASSWORD=... \
    MONGO_AUTH_TEST_OTHER_USER=... MONGO_AUTH_TEST_OTHER_PASSWORD=... \
    go test -run TestConnect_Auth ./pkg/db/
