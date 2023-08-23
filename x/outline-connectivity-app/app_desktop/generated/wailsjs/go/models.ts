export namespace shared_backend {
	
	export class CallInputMessage {
	    method: string;
	    input: string;
	
	    static createFrom(source: any = {}) {
	        return new CallInputMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.method = source["method"];
	        this.input = source["input"];
	    }
	}
	export class CallOutputMessage {
	    result: string;
	    errors: string[];
	
	    static createFrom(source: any = {}) {
	        return new CallOutputMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.result = source["result"];
	        this.errors = source["errors"];
	    }
	}

}

