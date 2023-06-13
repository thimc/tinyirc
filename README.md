tinyirc
=======

tinyirc is a simple IRC client written in go with no third party dependencies.

It is heavily inspired by suckless [sic](https://tools.suckless.org/sic/)
and it has been a really fun exercise for me to learn more about the
[net](https://pkg.go.dev/net) package.

Much like sic, tinyirc will read commands from standard input and print
everything to standard output. The data is multiplexed and so all traffic
is merged into one output.

Most of the scripts made for sic should work with tinyirc.

Installation
============

    $ make
    # make install


## Flags

The following flags are supported:

- `h` sets the IRC Host. Default is `irc.libera.chat`
- `p` sets the IRC Port. Defaults to `6667`
- `n` sets the user nickname. Defaults to the `$USER` variable
- `p` sets the user password
- `P` sets the command prefix, defaults to `/`
