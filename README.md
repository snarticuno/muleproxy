## Muleproxy

`muleproxy` is a proxy for `muledump` that:

* avoids the rate limit problem
* lets you put your muledump online without exposing your password

### Running Locally

If you just want to run `muledump` on your computer and not put it online, do this:

* Use the patched `muldeump` at https://github.com/killring/muledump
* Build `muleproxy` with `go build`
* Put `muleproxy` in the `muldeump` directory
* Configure `muledump` as normal, but don't put your password in `accounts.js`
* Configure `muleproxy`. Check out the example config
* Start `muleproxy` with `./muleproxy config.json`
* Go to http://localhost:5353/muledump.html in your browser
