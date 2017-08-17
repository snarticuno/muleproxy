## Muleproxy

`muleproxy` is a proxy for `muledump` that:

* avoids the rate limit problem
* lets you put your muledump online without exposing your password

You have to run this somewhere and modify muledump to use it instead of yql.

### Configure muleproxy

Build it with `go build`

Check out `example_config.js`. Set up a file like that with your account info.

Run the proxy with `muleproxy config.js`.

### Configure muledump

Patch muledump to use your proxy.

``` diff
--- a/lib/realmapi.js
+++ b/lib/realmapi.js
@@ -1,9 +1,6 @@
 (function($, window) {
 
-var BASEURL = [
-       'https://realmofthemadgodhrd.appspot.com/',
-       'https://rotmgtesting.appspot.com/'
-][+!!window.testing]
+var BASEURL = 'http://localhost:5353/';
 
 var _cnt = 0;
 function queue_request(obj) {
@@ -50,13 +47,8 @@ function realmAPI(path, opts, extraopts, callback) {
        }
 
        queue_request({
-               dataType: 'jsonp',
-               url: 'https://query.yahooapis.com/v1/public/yql',
-               data: {
```

In muledump's `accounts.js` put the md5 hash of your email where your
email would normally go. It doesn't matter what you put where the
password goes, just don't put your password.
