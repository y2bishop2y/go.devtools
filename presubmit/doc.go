// This file was auto-generated via go generate.
// DO NOT UPDATE MANUALLY

/*
The presubmit tool performs various presubmit related functions.

Usage:
   presubmit [flags] <command>

The presubmit commands are:
   query       Query open CLs from Gerrit
   post        Post review with the test results to Gerrit
   test        Run tests for a CL
   version     Print version
   help        Display help for commands or topics
Run "presubmit help [command]" for command usage.

The presubmit flags are:
   -host=: The Jenkins host. Presubmit will not send any CLs to an empty host.
   -netrc=/var/veyron/.netrc: The path to the .netrc file that stores Gerrit's credentials.
   -token=: The Jenkins API token.
   -url=https://veyron-review.googlesource.com: The base url of the gerrit instance.
   -v=false: Print verbose output.

Presubmit Query

This subcommand queries open CLs from Gerrit, calculates diffs from the previous
query results, and sends each one with related metadata (ref, repo, changeId) to
a Jenkins project which will run tests against the corresponding CL and post review
with test results.

Usage:
   presubmit query [flags]

The query flags are:
   -log_file=/var/veyron/tmp/presubmit_log: The file that stores the refs from the previous Gerrit query.
   -project=veyron-presubmit-test: The name of the Jenkins project to add presubmit-test builds to.
   -query=(status:open -project:experimental): The string used to query Gerrit for open CLs.

Presubmit Post

This subcommand posts review with the test results to Gerrit.

Usage:
   presubmit post [flags]

The post flags are:
   -msg=: The review message to post to Gerrit.
   -ref=: The ref where the review is posted.

Presubmit Test

This subcommand pulls the open CLs from Gerrit, runs tests specified in a config
file, and posts test results back to the corresponding Gerrit review thread.

Usage:
   presubmit test [flags]

The test flags are:
   -build_number=-1: The number of the Jenkins build.
   -conf=$VEYRON_ROOT/tools/conf/presubmit: The config file for presubmit tests.
   -manifest=manifest/v1/default: Name of the project manifest.
   -ref=: The ref where the review is posted.
   -repo=: The URL of the repository containing the CL pointed by the ref.
   -tests_base_path=$VEYRON_ROOT/scripts/jenkins: The base path of all the test scripts.

Presubmit Version

Print version of the presubmit tool.

Usage:
   presubmit version

Presubmit Help

Help with no args displays the usage of the parent command.
Help with args displays the usage of the specified sub-command or help topic.
"help ..." recursively displays help for all commands and topics.

Usage:
   presubmit help [flags] [command/topic ...]

[command/topic ...] optionally identifies a specific sub-command or help topic.

The help flags are:
   -style=text: The formatting style for help output, either "text" or "godoc".
*/
package main
