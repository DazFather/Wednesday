/**
 * Custom error representing a failed HTTP response (non-2xx).
 * Copies all properties from the original Response object.
 */
export class HttpError extends Error {
	constructor(response) {
		super("Invalid HTTP status code: " + response.statusText)
		Object.assign(this, response)
	}
}

/**
 * Lightweight, customizable HTTP client wrapper around the Fetch API.
 */
export class HttpClient {

	#opt = {
		method: "GET",
		headers: {
			"Accept": "application/json"
		},
		url: location.href,
		mode: "cors",
		credentials: "same-origin"
	}

	/**
	 * Get the current base URL.
	 * @returns {string}
	 */
	get basepath() {
		return this.#opt.url
	}


	/**
	 * Set default client options.
	 * This replaces defaults and merges headers.
	 * Also extracts preprocess/postprocess functions if present.
	 * @param {Object} [options={}] - Optional base configuration.
	 * @param {string} [options.url] - Base URL to resolve relative paths.
	 * @param {Object} [options.headers] - Default headers.
	 * @param {string} [options.method] - Default HTTP method (e.g., GET).
	 * @param {string} [options.mode] - Request mode (`cors`, `same-origin`, etc.).
	 * @param {string} [options.credentials] - Request credentials mode.
	 * @param {Function} [options.preprocess] - Custom function to process request options before sending.
	 * @param {Function} [options.postprocess] - Custom function to transform the response.
	 */
	set options(opt) {
		if (!opt || typeof opt !== "object") {
			throw new TypeError("Invalid options. Must be a valid object");
		}
		
		if (opt.preprocess) {
			this.preprocess = opt.preprocess
			delete opt.preprocess
		}
		if (opt.postprocess) {
			this.postprocess = opt.postprocess
			delete opt.postprocess
		}
		
		const std = this.options
		this.#opt = { 
			...std,
			...opt,
			headers: { ...std.headers, ...(opt?.headers ?? {}) }
		}
		this.#opt.url = this.#opt.url.replace(/\/?$/, "/");
	}

	/**
	 * Clone of the current request options.
	 * @returns {RequestInit & { url: string }}
	 */
	get options() {
		return structuredClone(this.#opt)
	}

	/**
	 * Lightweight, customizable HTTP client wrapper around the Fetch API.
	 */	
	timeout = 10 * 1000;

	#preprocess = req => {
		if (req.body && Object.getPrototypeOf(req.body) === Object.prototype) {
			req.body = JSON.stringify(req.body);
			req.headers = Object.assign({"Content-Type": "application/json"}, req.headers)
		}
		return req;
	};

	/**
	 * Set a custom request preprocessor function.
	 * @param {(req: RequestInit) => RequestInit} fn
	 */
	set preprocess(fn) {
		if (typeof fn !== "function") throw new TypeError("Invalid preprocessor. Must be a function");

		this.#preprocess = fn
	}

	#postprocess = res => res.json();

	/**
	 * Set a custom response postprocessor function.
	 * @param {(res: Response) => any | Promise<any>} fn
	 */
	set postprocess(fn) {
		if (typeof fn !== "function") throw new TypeError("Invalid postprocessor. Must be a function");

		this.#postprocess = fn
	}

	/**
	 * Create a new HttpClient instance.
	 * @param {Object} [options={}] - Optional base configuration.
	 * @param {string} [options.url] - Base URL to resolve relative paths.
	 * @param {Object} [options.headers] - Default headers.
	 * @param {string} [options.method] - Default HTTP method (e.g., GET).
	 * @param {string} [options.mode] - Request mode (`cors`, `same-origin`, etc.).
	 * @param {string} [options.credentials] - Request credentials mode.
	 * @param {Function} [options.preprocess] - Custom function to process request options before sending.
	 * @param {Function} [options.postprocess] - Custom function to transform the response.
	 */
	constructor(base = {}) {
		this.options = base
	}

	/**
	 * Resolve a URL against the base path.
	 * @param {string} input - Relative or absolute URL.
	 * @returns {string} - Absolute resolved URL.
	 */
	resolveUrl(input) {
		try {
			return new URL(input).href;
		} catch {
			return new URL(input, this.basepath).href;
		}
	}

	#genRequest(url, opt = {}) {
		let req = this.options
		let { headers } = req
		if (!!opt.headers) headers = { ...headers, ...opt.headers }

		req = { ...req, ...opt, headers }
		if (typeof req.preprocess === "function") {
			req = req.preprocess(req)
			delete req.preprocess
		} else if (this.#preprocess) {
			req = this.#preprocess(req)
		}

		return new Request(this.resolveUrl(url), req)
	}

	/**
	 * Perform a low-level fetch with all custom logic (timeout, preprocess, postprocess).
	 * @param {string} url - The request URL (can be relative).
	 * @param {RequestInit & { postprocess?: Function, preprocess?: Function }} [opt={}] - Fetch options.
	 * @returns {Promise<any>} - The processed response.
	 */
	async fetch(url = "", opt = {}) {
		let timer = null;
		if (this.timeout && !opt.signal) {
			const controller = new AbortController();
			opt.signal = controller.signal;
			timer = setTimeout(() => controller.abort(), this.timeout);
		}
		const postprocess = opt.postprocess ?? this.#postprocess

		const req = this.#genRequest(url, opt)

		try {
			const res = await fetch(req);
			if (!res.ok) throw new HttpError(res);

			if (!!postprocess) {
				const out = postprocess(res)
				return out instanceof Promise ? await out : out
			}
			return res
		} finally {
			if (timer) clearTimeout(timer);
		}
	}

	/**
	 * Send a request using the given HTTP method.
	 * @param {string} verb - HTTP method (GET, POST, PUT, etc.).
	 * @param {string} url - Request path or URL.
	 * @param {any} [body=null] - Optional request body.
	 * @returns {Promise<any>}
	 */
	async send(verb = this.#opt.method ?? "GET", url = "", body = null) {
		return this.fetch(url, { method: verb, body })
	}

	/**
	 * Shortcut for `GET` request.
	 * @param {string} path - Path or URL to request.
	 * @returns {Promise<any>}
	 */
	get(path) {
		return this.send("GET", path);
	}

	/**
	 * Fetch a file and optionally trigger download if `Content-Disposition` is `attachment`.
	 * @param {string} path - File URL or path.
	 * @param {string} filename - Fallback filename if not provided in headers.
	 * @returns {Promise<File>}
	 */
	getFile(path, filename) {
		const postprocess = async res => {
			const blob = await res.blob();
			const disp = res.headers.get("Content-Disposition")
			
			if (disp) {
				let rgx = /filename=(["']?)(.+)\1/gi
				let match = rgx.exec(disp)
				if (match) filename = match[2]

				rgx = /(?:disposition=(["']?)(\w+)\1)|(?:^(\w+))/i;
				match = disp.match(rgx)
				switch (match?.[2] || match?.[3] || null) {
				case "attachment":
					const url = URL.createObjectURL(blob);
					const a = document.createElement("a");
					a.href = url;
					a.download = filename
					document.body.appendChild(a);
					a.click();
					document.body.removeChild(a);
					URL.revokeObjectURL(url);
					break;
				case "inline":
				case null:
					break;
				default:
					console.error("Invalid disposition header:", disp)
				}
			}

			return new File([blob], filename, { type: blob.type, lastModified: Date.now() })
		}
		return this.fetch(path, { method: "GET", postprocess })
	}

	/**
	 * Shortcut for `POST` request.
	 * @param {string} path - URL path.
	 * @param {any} body - Request body.
	 * @returns {Promise<any>}
	 */
	post(path, body) {
		return this.send("POST", path, body);
	}

	/**
	 * Shortcut for `POST` request.
	 * @param {string} path - URL path.
	 * @param {any} body - Request body.
	 * @returns {Promise<any>}
	 */
	put(path, body) {
		return this.send("PUT", path, body);
	}

	/**
	 * Shortcut for `POST` request.
	 * @param {string} path - URL path.
	 * @param {any} body - Request body.
	 * @returns {Promise<any>}
	 */
	patch(path, body) {
		return this.send("PATCH", path, body);
	}

	/**
	 * Shortcut for `POST` request.
	 * @param {string} path - URL path.
	 * @param {any} body - Request body.
	 * @returns {Promise<any>}
	 */
	delete(path, body = null) {
		return this.send("DELETE", path, body);
	}

}

/**
 * @typedef {Object} FragmentHandler
 * @property {() => Promise<DocumentFragment>} clone - Async function that returns a new clone of the fetched resource.
 * @property {(target: Target) => Promise<Node>} appendTo - Async function that appends a new clone of the resource as the last child of the specified target
 * @property {(target: Target, before?: Target) => Promise<Node>} insertTo - Async function that inserts a new clone of the resource on the specified target before another.
 * @property {(target: Target) => Promise<Node>} replace - Async function that replaces the specified target with a new clone of the resource.
 */

/**
 * Fetches an HTML document or fragment and returns reusable DOM insertion helpers.
 * @param {string} url - The URL to fetch.
 * @param {RequestInit} [opt] - Optional fetch options.
 * @param {(fragment: DocumentFragment) => void} [init] - Optional callback to mutate each clone.
 * @returns {FragmentHandler}
 */
export function useFragment(url, opt = {}, init = null) {
	const cached = null 
	
	const load = async () => {
		if (cached) return cached

		const response = await fetch(url, opt);
		if (!response.ok) throw new HttpError(response)

		const htmlText = await response.text();

		const content = document.createElement("div");
		content.innerHTML = htmlText

		const body = content.querySelector("body")
			?? content.querySelector("html")
			?? content

		const fragment = new DocumentFragment();
		for (let node of body.childNodes) {
			fragment.appendChild(node.cloneNode(true))
		}

		return cached = fragment;
	}

	const clone = !init
		? async () => await laod()
		: async () => {
			const frag = await load();
			init(frag);
			return frag;
		};

	return {
		clone,
		appendTo: async (target) => select(target).appendChild(await clone()),
		insertTo: async (target, before = null) =>
			select(target).insertBefore(await clone(), before ? select(before) : null),
		replace: async (target) => select(target).replaceWith(await clone()),
	};
}

