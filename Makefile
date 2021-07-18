PKG_FILES=`go list ./... | sed -e 's=github.com/git-hulk/routines/=./='`

CCCOLOR="\033[37;1m"
MAKECOLOR="\033[32;1m"
ENDCOLOR="\033[0m"

all:

.PHONY: all

test:
	@go test -v -covermode=count -coverprofile=coverage.out

race:
	@go test -v -race

lint:
	@rm -rf lint.log
	@printf $(CCCOLOR)"Checking format...\n"$(ENDCOLOR)
	@go list ./... | sed -e 's=github.com/git-hulk/routines=.=' | xargs -n 1 gofmt -d -s 2>&1 | tee -a lint.log
	@[ ! -s lint.log ]
	@printf $(CCCOLOR)"Checking import...\n"$(ENDCOLOR)
	@go list ./... | sed -e 's=github.com/git-hulk/routines=.=' | xargs -n 1 goimports -d 2>&1 | tee -a lint.log
	@[ ! -s lint.log ]
	@printf $(CCCOLOR)"Checking vet...\n"$(ENDCOLOR)
	@go list ./... | sed -e 's=github.com/git-hulk/rotuines=.=' | xargs -n 1 go vet 2>&1 | tee -a lint.log
	@[ ! -s lint.log ]

