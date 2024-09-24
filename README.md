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
```
