# Autopilot

Use this to generate a cannonical trace of what the API actually does.
This will use a set of small files that we keep in git to run against
an executing local server.

```bash
  go run autopilot.go > autopilot.md
```

This markdown file should be posted in the object-drive-server wiki.
Over time, the main path for this function should produce a
substantial legible document that removes any ambiguities in the
reference guide.

# Creating Scenarios

This code can be used to script scenarios against the real server that
will produce http traces.  It can also be used to drive scenarios against
real servers, such as the deployed EC2 demo, for the purposes of simulating
a larger user population and filling the server with data.

If a scenario which uploads large binary files is to be undertaken, please
do not check the upload directories into git.  Ensure that the home directories
for the users gets set to something that is not under our $GOPATH anywhere.

