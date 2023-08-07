export namespace main {
	
	export class ConnectivityTestError {
	    operation: string;
	    posixError: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ConnectivityTestError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.operation = source["operation"];
	        this.posixError = source["posixError"];
	        this.message = source["message"];
	    }
	}
	export class ConnectivityTestProtocolConfig {
	    tcp: boolean;
	    udp: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ConnectivityTestProtocolConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tcp = source["tcp"];
	        this.udp = source["udp"];
	    }
	}
	export class ConnectivityTestResult {
	    proxy: string;
	    resolver: string;
	    proto: string;
	    prefix: string;
	    // Go type: time
	    time: any;
	    durationMs: number;
	    error?: ConnectivityTestError;
	
	    static createFrom(source: any = {}) {
	        return new ConnectivityTestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.proxy = source["proxy"];
	        this.resolver = source["resolver"];
	        this.proto = source["proto"];
	        this.prefix = source["prefix"];
	        this.time = this.convertValues(source["time"], null);
	        this.durationMs = source["durationMs"];
	        this.error = this.convertValues(source["error"], ConnectivityTestError);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

