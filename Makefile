include $(GOROOT)/src/Make.inc

TARG=rangehdr

GOFILES=\
	rangehdr.go

GOTESTFILES=\
	rangehdr_test.go

include $(GOROOT)/src/Make.pkg
