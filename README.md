# Pubsub Function

This is cloud function project built for Pulsar.

### Development set up
The steps how to start a single function worker process.
```bash
$ cd src
$ go run main.go
```

### Support of Javascript function
A trigger function must be implemented with http request and response as function paramters. These are the same as http reponse and request object. An example is at [function-pack folder](function-pack/js/example-funtion.js)

The response body from the trigger function is passed on to an output topic.

```
function trigger(req, res) {
    const data = {attr: true}
    res.statusCode=202
    res.end(JSON.stringify({attr: true, attr1: "somemessage"})) // this can be passed to an output topic

}

exports.trigger = trigger;
```

### Function registration
The function registation including uploading the javascript file is done by http multi-form-data upload. 

##### cURL example
```
curl --location --request POST 'localhost:8081/v2/function/ming-luo/testfunction' \
--header 'Authorization: Bearer Pulsar-JWT' \
--form 'source=@/home/ming/go/src/github.com/kafkaesque-io/pubsub-function/function-pack/js/ming.js' \
--form 'language-pack=js' \
--form 'parallelism=1' \
--form 'trigger-type=pulsar-topic' \
--form 'function-status=activated' \
--form 'input-topic=persistent://ming-luo/local-useast1-gcp/test-topic' \
--form 'output-topic=persistent://ming-luo/local-useast1-gcp/test-topic2'
```

##### Node
```
var unirest = require('unirest');
var req = unirest('POST', 'localhost:8081/v2/function/ming-luo/testfunction')
  .headers({
    'Authorization': 'Bearer pulsar-jwt'
  })
  .attach('file', '/home/ming/go/src/github.com/kafkaesque-io/pubsub-function/function-pack/js/ming.js')
  .field('language-pack', 'js')
  .field('parallelism', '1')
  .field('trigger-type', 'pulsar-topic')
  .field('function-status', 'activated')
  .field('input-topic', 'persistent://ming-luo/local-useast1-gcp/test-topic')
  .field('output-topic', 'persistent://ming-luo/local-useast1-gcp/test-topic2')
  .end(function (res) { 
    if (res.error) throw new Error(res.error); 
    console.log(res.raw_body);
  });
```