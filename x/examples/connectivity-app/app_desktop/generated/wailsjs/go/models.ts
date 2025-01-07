export namespace shared_backend {
	
	export class Response {
	    body: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new Response(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.body = source["body"];
	        this.error = source["error"];
	    }
	}

}

