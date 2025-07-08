(function (global) {
    var window = global;
    global.self = window;
    global.window = window;
    global.top = window;
    global.parent = window;

    global.location = {
        href: "{{.Href}}",
        protocol: "{{.Scheme}}:",
        host: "{{.Host}}",
        hostname: "{{.Hostname}}",
        port: "{{.Port}}",
        pathname: "{{.Path}}",
        search: "?{{.RawQuery}}",
        hash: "{{.Fragment}}",
        assign: function() {},
        reload: function() {},
        replace: function() {}
    };

    global.navigator = {
        userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
        platform: "Win32",
        language: "en-US"
    };

    global.history = {
        replaceState: function() {}
    };

    global.document = {
        elements: {},
        getElementById: function(id) {
            if (!this.elements[id]) { this.elements[id] = { value: "" }; }
            return this.elements[id];
        },
        createElement: function(tag) {
            return { setAttribute: function() {}, src: "" };
        },
        getElementsByTagName: function(name) {
            if (name === 'head' || name === 'body') {
                return [{
                    appendChild: function(element) {
                        // CRITICAL: Intercept and neutralize dynamic script loading.
                    }
                }];
            }
            return [];
        },
        cookie: ""
    };

    var atobImpl = 'var chars="ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=";var a,b,c,d,e,f,g,i=0,result="";str=str.replace(/[^A-Za-z0-9\\+\\/\\=]/g,"");do{a=chars.indexOf(str.charAt(i++));b=chars.indexOf(str.charAt(i++));c=chars.indexOf(str.charAt(i++));d=chars.indexOf(str.charAt(i++));e=a<<18|b<<12|c<<6|d;f=e>>16&255;g=e>>8&255;a=e&255;result+=String.fromCharCode(f);if(c!=64)result+=String.fromCharCode(g);if(d!=64)result+=String.fromCharCode(a)}while(i<str.length);return result;';
    global.atob = new Function('str', atobImpl);

})(typeof globalThis !== 'undefined' ? globalThis : this);