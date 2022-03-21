all:
	(cd x11link-server; go build)
	(cd x11link-client; go build)
clean:
	rm -f x11link-server/x11linke-server
	rm -f x11link-client/x11linke-client
