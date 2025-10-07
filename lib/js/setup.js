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
        forms: [],
        getElementById: function(id) {
            if (!this.elements[id]) { 
                this.elements[id] = { 
                    value: "",
                    setAttribute: function() {},
                    getAttribute: function() { return ""; },
                    style: {},
                    className: "",
                    innerHTML: "",
                    textContent: "",
                    appendChild: function() {},
                    removeChild: function() {},
                    click: function() {},
                    focus: function() {},
                    blur: function() {}
                }; 
            }
            return this.elements[id];
        },
        createElement: function(tag) {
            return { 
                setAttribute: function() {}, 
                getAttribute: function() { return ""; },
                src: "",
                style: {},
                className: "",
                innerHTML: "",
                textContent: "",
                appendChild: function() {},
                removeChild: function() {},
                click: function() {},
                tagName: tag.toUpperCase(),
                firstChild: { href: global.location.href }
            };
        },
        getElementsByTagName: function(name) {
            if (name === 'head' || name === 'body') {
                return [{
                    appendChild: function(element) {
                        // CRITICAL: Intercept and neutralize dynamic script loading.
                    }
                }];
            }
            if (name === 'form') {
                return this.forms;
            }
            return [];
        },
        getElementsByClassName: function(className) {
            return [];
        },
        querySelector: function(selector) {
            return null;
        },
        querySelectorAll: function(selector) {
            return [];
        },
        cookie: "",
        readyState: "complete",
        addEventListener: function() {},
        removeEventListener: function() {},
        body: {
            appendChild: function() {},
            removeChild: function() {},
            style: {}
        },
        head: {
            appendChild: function() {},
            removeChild: function() {}
        }
    };

    var atobImpl = 'var chars="ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=";var a,b,c,d,e,f,g,i=0,result="";str=str.replace(/[^A-Za-z0-9\\+\\/\\=]/g,"");do{a=chars.indexOf(str.charAt(i++));b=chars.indexOf(str.charAt(i++));c=chars.indexOf(str.charAt(i++));d=chars.indexOf(str.charAt(i++));e=a<<18|b<<12|c<<6|d;f=e>>16&255;g=e>>8&255;a=e&255;result+=String.fromCharCode(f);if(c!=64)result+=String.fromCharCode(g);if(d!=64)result+=String.fromCharCode(a)}while(i<str.length);return result;';
    global.atob = new Function('str', atobImpl);
    
    // Add btoa as well
    global.btoa = function(str) {
        var chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=';
        var result = '';
        var i = 0;
        while (i < str.length) {
            var a = str.charCodeAt(i++);
            var b = i < str.length ? str.charCodeAt(i++) : 0;
            var c = i < str.length ? str.charCodeAt(i++) : 0;
            var bitmap = (a << 16) | (b << 8) | c;
            result += chars.charAt((bitmap >> 18) & 63);
            result += chars.charAt((bitmap >> 12) & 63);
            result += chars.charAt((bitmap >> 6) & 63);
            result += chars.charAt(bitmap & 63);
        }
        var padding = str.length % 3;
        return padding ? result.slice(0, padding - 3) + '==='.substring(padding) : result;
    };

    // Add common timer functions that might be needed
    global.setTimeout = global.setTimeout || function(fn, delay) {
        // For synchronous execution, just call immediately
        if (typeof fn === 'function') {
            fn();
        }
        return 1;
    };
    
    global.clearTimeout = global.clearTimeout || function() {};
    global.setInterval = global.setInterval || function() { return 1; };
    global.clearInterval = global.clearInterval || function() {};

})(typeof globalThis !== 'undefined' ? globalThis : this);