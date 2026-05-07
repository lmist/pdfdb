export namespace doccache {

	export class Health {
	    ready: boolean;
	    cached: number;
	    total: number;
	    cacheDir: string;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new Health(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ready = source["ready"];
	        this.cached = source["cached"];
	        this.total = source["total"];
	        this.cacheDir = source["cacheDir"];
	        this.message = source["message"];
	    }
	}

}

export namespace main {

	export class DocumentState {
	    id: string;
	    slug: string;
	    title: string;
	    filename: string;
	    sourceUrl?: string;
	    sizeBytes: number;
	    pageCount: number;
	    open: boolean;

	    static createFrom(source: any = {}) {
	        return new DocumentState(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.slug = source["slug"];
	        this.title = source["title"];
	        this.filename = source["filename"];
	        this.sourceUrl = source["sourceUrl"];
	        this.sizeBytes = source["sizeBytes"];
	        this.pageCount = source["pageCount"];
	        this.open = source["open"];
	    }
	}
	export class State {
	    profiles: profiles.Profile[];
	    documents: DocumentState[];
	    open: zathura.OpenDocument[];
	    health: doccache.Health;
	    needsDb: boolean;
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new State(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profiles = this.convertValues(source["profiles"], profiles.Profile);
	        this.documents = this.convertValues(source["documents"], DocumentState);
	        this.open = this.convertValues(source["open"], zathura.OpenDocument);
	        this.health = this.convertValues(source["health"], doccache.Health);
	        this.needsDb = source["needsDb"];
	        this.error = source["error"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
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

export namespace profiles {

	export class Profile {
	    name: string;
	    active: boolean;

	    static createFrom(source: any = {}) {
	        return new Profile(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.active = source["active"];
	    }
	}

}

export namespace zathura {

	export class OpenDocument {
	    slug: string;
	    pid: number;
	    path: string;

	    static createFrom(source: any = {}) {
	        return new OpenDocument(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slug = source["slug"];
	        this.pid = source["pid"];
	        this.path = source["path"];
	    }
	}

}
