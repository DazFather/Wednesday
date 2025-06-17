/**
 * A string selector to an element of the DOM or an Element (commonly an HTMLElement).
 * @typedef {(string|Element)} Target
 */

/**
 * Selects an element from the DOM. Used in other functions for flexibility.
 * @param {Target} target - The target to select.
 * @return {Element} The selected element.
 */
function select(target) {
	return typeof target === "string" ? document.querySelector(target) : target;
}

/**
 * @callback MutatePropFn
 * @param {?} value - The property value.
 * @param {string} [prop] - The property name.
 * @return {*} The mutated value.
 */

/**
 * Creates a mirror object with access to specified properties of the target.
 * @param {Target} target - The target to select.
 * @param {(string|string[])} props - One or more properties of the target accessible through the mirror.
 * @param {MutatePropFn} [mutate] - Optional callback to mutate the value before being set.
 * @return {object} The mirror object.
 */
function useMirror(target, props, mutate) {
	const elem = select(target);
	if (typeof props === "string") props = [props];

	return props.reduce((obj, prop) => {
		Object.defineProperty(obj, prop, {
			get() {
				return elem[prop];
			},
			set(value) {
				if (!!mutate) value = mutate(value, prop);
				elem[prop] = value;
			},
		});
		return obj;
	}, {});
}

/**
 * Generates a function for editing the innerText of an element.
 * @param {Target} target - The target to select.
 * @param {(value: ?, elem: Element) => ?} [mutate] - Optional callback to modify the text before updating innerText.
 * @return {(text: string) => string} A function to edit the target's innerText.
 */
function useDisplay(target, mutate) {
	const elem = select(target);

	return !!mutate
		? (text) => (elem.innerText = mutate(text, elem))
		: (text) => (elem.innerText = text);
}

/**
 * @typedef {object} TemplateHandler
 * @property {() => Node} clone - Returns a new clone of the template.
 * @property {(target: Target) => Node} appendTo - Appends a new clone of the template as the last child of the specified target.
 * @property {(target: Target, before?: Target) => Node} insertTo - Inserts a new clone of the template on the specified target before another.
 * @property {(target: Target) => Node} replace - Replaces the specified target with a new clone of the template.
 */

/**
 * Generates an object with methods to manage the provided template.
 * @param {string} templateID - The DOM id of the template to use.
 * @param {(cloned: Element) => ?} [mutate] - Optional callback to initialize the template on each clone.
 * @return {TemplateHandler} Utility methods for handling the template.
 */
function useTemplate(templateID, init) {
	const templ = document.getElementById(templateID);
	const clone = !init
		? () => templ.content.cloneNode(true)
		: () => {
			const elem = templ.content.cloneNode(true);
			init(elem);
			return elem;
		};

	return {
		clone,
		appendTo: (target) => select(target).appendChild(clone()),
		insertTo: (target, before = null) =>
			select(target).insertBefore(clone(), select(before)),
		replace: (target) => select(target).replaceWith(clone()),
	};
}

/**
 * Binds properties between elements and an object, updating them in response to events.
 * The element to bind must have the "bind" attribute set.
 * @param {Target} target - A target to search for bound elements, typically a common parent.
 * @param {MutatePropFn} [mutate] - Optional callback to change a property value before setting it.
 * @return {object} An object with bound properties.
 */
function useBinds(target, mutate) {
	const parent = select(target);
	const res = {};

	const bindMap = Array.from(parent.querySelectorAll("[bind]")).reduce(
		(obj, elem) => {
			elem.contenteditable = "true";
			elem
				.getAttribute("bind")
				.match(/([\w:]+)/g)
				.forEach((attr) => {
					let [prop, name, eventName] = attr.split(":");
					if (!name) name = prop;

					if (!!eventName)
						elem.addEventListener(eventName, (e) => (res[name] = elem[prop]));

					let proplist = obj[name];
					if (!proplist) obj[name] = proplist = [];

					proplist.push({ prop, elem, prev: elem[prop] });
				});
			return obj;
		},
		{},
	);

	for (let key in bindMap) {
		Object.defineProperty(res, key, {
			get() {
				const item = bindMap[key][0];
				return item.elem[item.prop];
			},
			set(value) {
				if (!!mutate) value = mutate(value, key);
				bindMap[key].forEach((item) => (item.elem[item.prop] = value));
			},
		});
		res[key] = res[key];
	}

	return res;
}

/**
 * Generates an object with a value property, triggering the provided function when modified.
 * @param {?} start - Initial state for the effect.
 * @param {(newValue: ?, oldValue: ?) => ?} onChange - Callback to run each time value is updated.
 * @return {{value: *}} An object with a value property, triggering the onChange function.
 */
function useEffect(start, onChange) {
	let state = start;

	return {
		get value() {
			return state;
		},
		set value(v) {
			state = onChange(v, state);
		},
	};
}

/**
 * Access and modify resources between text/javascript and module across the page.
 * @param {?} obj - Any resources to pass. Objects (including array) will be unpacked, all others will replace bridge()[0]
 * @return {object} Storage object (reference will never change).
 */
function bridge(obj) {
	if (arguments.length === 0)
		return window._wed_bridge ?? (window._wed_bridge = [])

	if (typeof obj != "object") obj = [obj]

	return !window._wed_bridge
		? window._wed_bridge = Object.assign([], obj)
		: Object.assign(window._wed_bridge, {...obj })
}

// make funciton available accross all page (also used to re-import them via ECMAScript module when needed)
window._wed_utility = {
	select,
	useMirror,
	useDisplay,
	useTemplate,
	useBinds,
	useEffect,
	bridge
}
