# Common libraries

## How to add new library

1. Create folder ./<name>
2. write down a lot of code in ./<name>
3. cd ./<name>
4. go mod init github.com/RRWM1rr0rB/faraway_lib/backend/golang/<name>
5. go mod tidy
6. add to Makefile to variable `NAMES` your new library name
7. GOPROXY=direct GOPRIVATE=github.com/* go get -u ggithub.com/RRWM1rr0rB/faraway_lib/backend/golang/errors