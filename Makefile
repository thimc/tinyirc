NAME = tinyirc
PREFIX = /usr/local
VERSION = 0.1.1
BUILD_TIME = `date -u '+%Y-%m-%dT%H:%M:%SZ'`

${NAME}:
	go build -o $(NAME) -ldflags \
		"-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" main.go

clean:
	rm -f ${NAME}

install: ${NAME}
	cp -f ${NAME} "${DESTDIR}${PREFIX}/bin"
	cp -f ${NAME}.1 "${DESTDIR}${PREFIX}/man/man1/"
	chmod 755 "${DESTDIR}${PREFIX}/bin/${NAME}"

uninstall:
	rm -f "${DESTDIR}${PREFIX}/bin/${NAME}"
	rm -f "${DESTDIR}${PREFIX}/man/man1/${NAME}.1"

.PHONY: clean install uninstall
