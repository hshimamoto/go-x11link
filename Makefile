dirs := x11link-server x11link-client
suffix =
ifneq ($(GOOS),)
	export GOOS
endif
ifneq ($(GOARCH),)
	export GOARCH
	suffix = .$(GOARCH)
endif

all: $(addprefix _dir_, $(dirs))

.PHONY: $(addprefix _dir_, $(dirs))
$(addprefix _dir_, $(dirs)):
	(cd $(patsubst _dir_%, %, $@); go build -o $(patsubst _dir_%, %, $@)$(suffix))

clean: $(addprefix _clean_, $(dirs))

.PHONY: $(addprefix _clean_, $(dirs))
$(addprefix _clean_, $(dirs)):
	rm -f $(join $(patsubst _clean_%, %, $@)/, $(patsubst _clean_%, %, $@)*)
