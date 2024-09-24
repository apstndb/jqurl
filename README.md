# jqurl

Yet another curl & jq wrapper for me.

Execute `jq | curl | jq` in one command!

It is my hobby project and not for production use.

```sh

# Currently, `gojq` is required in $PATH
$ go install github.com/itchyny/gojq@latest

$ go install github.com/apstndb/jqurl@latest

# `--data-jq filter` is shorthand for `jq -n filter | curl --json @-`
# `--ofilter filter` is shorthand for `curl | jq filter`
$ jqurl -s --data-jq '{test: "hoge"}' https://httpbin.org/post --ofilter ".json"

# --auth=google adds Authorization header using ADC
# all opts in `--oopts opts` are passed to output jq
$ jqurl -s --auth=google https://oauth2.googleapis.com/tokeninfo --oopts '.scope -r'

# real World example: call Cloud Spanner's ExecuteSQL and show result set type
$ SPANNER_PROJECT=your-project SPANNER_INSTANCE=your-instance SPANNER_DATABASE=your-database
$ SESSION=$(jqurl -s --auth=google --data-jq '.session = {}' "https://spanner.googleapis.com/v1/projects/${SPANNER_PROJECT}/instances/${SPANNER_INSTANCE}/databases/${SPANNER_DATABASE}/sessions" --oopts=".name -r")
$ jqurl -s --auth=google "https://spanner.googleapis.com/v1/${SESSION}:executeSql" --data-jq '.transaction.singleUse = {} | .queryMode = "PLAN" | .sql = "SELECT 1 AS i"' --oopts='-c .metadata.rowType.fields'
[{"name":"i","type":{"code":"INT64"}}]
```
