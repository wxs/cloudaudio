#Cloudaudio

Course project for CSC2231: Cloud

## Running
Running the test server. Copy the code into your $GOPATH src tree. The binaries are in
the "srv" directory. Change into "srv" and run using

`go run server.go`

once the server has started running, you can test to make sure it's working by running
the mock input generator:

`go run mockinput.go`

Alternatively, you can load `http://localhost:2444/connect`, which should give you 
a session ID, or visit `http://localhost:2444/sessions` to see a list of active sessions




