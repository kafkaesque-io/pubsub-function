function trigger(req, res) {
    const data = {attr: true}
    res.statusCode=202
    res.end(JSON.stringify({attr: true, attr1: "asdf"}))

}

exports.trigger = trigger;
