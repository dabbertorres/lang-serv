# Lang-Serv
[![Build Status](https://travis-ci.org/dabbertorres/lang-serv.svg?branch=master)](https://travis-ci.org/dabbertorres/lang-serv)
[![Coverage Status](https://coveralls.io/repos/github/dabbertorres/lang-serv/badge.svg?branch=master)](https://coveralls.io/github/dabbertorres/lang-serv?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/dabbertorres/lang-serv)](https://goreportcard.com/report/github.com/dabbertorres/lang-serv)

A web server using Docker to provide an editor and runtime/compiler for any language that has an image on Docker Hub.

## How to
After running the server (`./lang-serv`), navigate to `{host}/{language}/{version}` in your browser!

#### Examples
* To get Python 3.6, navigate to: host-url.com/python/3.6.
* To get GCC 7.3, navigate to: host-url.com/gcc/7.3

## License
[MIT](LICENSE.md)
