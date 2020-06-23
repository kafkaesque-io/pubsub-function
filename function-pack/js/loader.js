/**
 * This is a function loader
 * 
 * $node loader.js <port> <path the function script>
 */
const http = require('http');

const cmdArgs = process.argv;

const port = cmdArgs[2];

const fn = require(cmdArgs[3])

console.log(port, fn)

//create a server object:
http.createServer(function (req, res) {
    const url = req.url;

    if (url === '/health') {
        res.statusCode = 200
        res.end()
    }
    if (url === '/kill') {
        console.log("the process is stopped as requested")
        process.exit(2)
    }

    fn.trigger(req, res);
    res.end();
}).listen(Number(port), function(){
    console.log("server start at port " + port); //the server object listens on port 3000
});