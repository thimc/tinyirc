NAME = tinyirc

PREFIX = /usr/local

${NAME}:
	go build -o ${NAME} main.go

clean:
	rm -f ${NAME}

install: ${NAME}
	cp -f ${NAME} "${DESTDIR}${PREFIX}/bin"
	cp -f ${NAME}.1 "${DESTDIR}${PREFIX}/man/man1/"
	chmod 755 "${DESTDIR}${PREFIX}/bin/${BIN}"

uninstall:
	rm -f "${DESTDIR}${PREFIX}/bin/${NAME}"
	rm -f "${DESTDIR}${PREFIX}/man/man1/${NAME}.1"

.PHONY: clean install uninstall
