var Module = (() => {
  var _scriptName = import.meta.url;

  return (
    async function (moduleArg = {}) {
      var moduleRtn;

      // include: shell.js
      // The Module object: Our interface to the outside world. We import
      // and export values on it. There are various ways Module can be used:
      // 1. Not defined. We create it here
      // 2. A function parameter, function(moduleArg) => Promise<Module>
      // 3. pre-run appended it, var Module = {}; ..generated code..
      // 4. External script tag defines var Module.
      // We need to check if Module already exists (e.g. case 3 above).
      // Substitution will be replaced with actual code on later stage of the build,
      // this way Closure Compiler will not mangle it (e.g. case 4. above).
      // Note that if you want to run closure, and also to use Module
      // after the generated code, you will need to define   var Module = {};
      // before the code. Then that object will be used in the code, and you
      // can continue to use Module afterwards as well.
      var Module = moduleArg;

      // Set up the promise that indicates the Module is initialized
      var readyPromiseResolve, readyPromiseReject;

      var readyPromise = new Promise((resolve, reject) => {
        readyPromiseResolve = resolve;
        readyPromiseReject = reject;
      });

      // Determine the runtime environment we are in. You can customize this by
      // setting the ENVIRONMENT setting at compile time (see settings.js).
      // Attempt to auto-detect the environment
      var ENVIRONMENT_IS_WEB = typeof window == "object";

      var ENVIRONMENT_IS_WORKER = typeof WorkerGlobalScope != "undefined";

      // N.b. Electron.js environment is simultaneously a NODE-environment, but
      // also a web environment.
      var ENVIRONMENT_IS_NODE = typeof process == "object" &&
        typeof process.versions == "object" &&
        typeof process.versions.node == "string" && process.type != "renderer";

      var ENVIRONMENT_IS_SHELL = !ENVIRONMENT_IS_WEB && !ENVIRONMENT_IS_NODE &&
        !ENVIRONMENT_IS_WORKER;

      if (ENVIRONMENT_IS_NODE) {
        // When building an ES module `require` is not normally available.
        // We need to use `createRequire()` to construct the require()` function.
        const { createRequire } = await import("module");
        /** @suppress{duplicate} */ var require = createRequire(
          import.meta.url,
        );
      }

      // --pre-jses are emitted after the Module integration code, so that they can
      // refer to Module (if they choose; they can also define Module)
      // include: lib/binding_web/lib/prefix.js
      Module.currentQueryProgressCallback = null;

      Module.currentProgressCallback = null;

      Module.currentLogCallback = null;

      Module.currentParseCallback = null;

      // end include: lib/binding_web/lib/prefix.js
      // Sometimes an existing Module object exists with properties
      // meant to overwrite the default module functionality. Here
      // we collect those properties and reapply _after_ we configure
      // the current environment's defaults to avoid having to be so
      // defensive during initialization.
      var moduleOverrides = {
        ...Module,
      };

      var arguments_ = [];

      var thisProgram = "./this.program";

      var quit_ = (status, toThrow) => {
        throw toThrow;
      };

      // `/` should be present at the end if `scriptDirectory` is not empty
      var scriptDirectory = "";

      function locateFile(path) {
        if (Module["locateFile"]) {
          return Module["locateFile"](path, scriptDirectory);
        }
        return scriptDirectory + path;
      }

      // Hooks that are implemented differently in different runtime environments.
      var readAsync, readBinary;

      if (ENVIRONMENT_IS_NODE) {
        // These modules will usually be used on Node.js. Load them eagerly to avoid
        // the complexity of lazy-loading.
        var fs = require("fs");
        var nodePath = require("path");
        // EXPORT_ES6 + ENVIRONMENT_IS_NODE always requires use of import.meta.url,
        // since there's no way getting the current absolute path of the module when
        // support for that is not available.
        if (!import.meta.url.startsWith("data:")) {
          scriptDirectory =
            nodePath.dirname(require("url").fileURLToPath(import.meta.url)) +
            "/";
        }
        // include: node_shell_read.js
        readBinary = (filename) => {
          // We need to re-wrap `file://` strings to URLs.
          filename = isFileURI(filename) ? new URL(filename) : filename;
          var ret = fs.readFileSync(filename);
          return ret;
        };
        readAsync = async (filename, binary = true) => {
          // See the comment in the `readBinary` function.
          filename = isFileURI(filename) ? new URL(filename) : filename;
          var ret = fs.readFileSync(filename, binary ? undefined : "utf8");
          return ret;
        };
        // end include: node_shell_read.js
        if (!Module["thisProgram"] && process.argv.length > 1) {
          thisProgram = process.argv[1].replace(/\\/g, "/");
        }
        arguments_ = process.argv.slice(2);
        // MODULARIZE will export the module in the proper place outside, we don't need to export here
        quit_ = (status, toThrow) => {
          process.exitCode = status;
          throw toThrow;
        };
      } // Note that this includes Node.js workers when relevant (pthreads is enabled).
      // Node.js workers are detected as a combination of ENVIRONMENT_IS_WORKER and
      // ENVIRONMENT_IS_NODE.
      else if (ENVIRONMENT_IS_WEB || ENVIRONMENT_IS_WORKER) {
        if (ENVIRONMENT_IS_WORKER) {
          // Check worker, not web, since window could be polyfilled
          scriptDirectory = self.location.href;
        } else if (typeof document != "undefined" && document.currentScript) {
          // web
          scriptDirectory = document.currentScript.src;
        }
        // When MODULARIZE, this JS may be executed later, after document.currentScript
        // is gone, so we saved it, and we use it here instead of any other info.
        if (_scriptName) {
          scriptDirectory = _scriptName;
        }
        // blob urls look like blob:http://site.com/etc/etc and we cannot infer anything from them.
        // otherwise, slice off the final part of the url to find the script directory.
        // if scriptDirectory does not contain a slash, lastIndexOf will return -1,
        // and scriptDirectory will correctly be replaced with an empty string.
        // If scriptDirectory contains a query (starting with ?) or a fragment (starting with #),
        // they are removed because they could contain a slash.
        if (scriptDirectory.startsWith("blob:")) {
          scriptDirectory = "";
        } else {
          scriptDirectory = scriptDirectory.slice(
            0,
            scriptDirectory.replace(/[?#].*/, "").lastIndexOf("/") + 1,
          );
        }
        {
          // include: web_or_worker_shell_read.js
          if (ENVIRONMENT_IS_WORKER) {
            readBinary = (url) => {
              var xhr = new XMLHttpRequest();
              xhr.open("GET", url, false);
              xhr.responseType = "arraybuffer";
              xhr.send(null);
              return new Uint8Array(/** @type{!ArrayBuffer} */ (xhr.response));
            };
          }
          readAsync = async (url) => {
            // Fetch has some additional restrictions over XHR, like it can't be used on a file:// url.
            // See https://github.com/github/fetch/pull/92#issuecomment-140665932
            // Cordova or Electron apps are typically loaded from a file:// url.
            // So use XHR on webview if URL is a file URL.
            if (isFileURI(url)) {
              return new Promise((resolve, reject) => {
                var xhr = new XMLHttpRequest();
                xhr.open("GET", url, true);
                xhr.responseType = "arraybuffer";
                xhr.onload = () => {
                  if (xhr.status == 200 || (xhr.status == 0 && xhr.response)) {
                    // file URLs can return 0
                    resolve(xhr.response);
                    return;
                  }
                  reject(xhr.status);
                };
                xhr.onerror = reject;
                xhr.send(null);
              });
            }
            var response = await fetch(url, {
              credentials: "same-origin",
            });
            if (response.ok) {
              return response.arrayBuffer();
            }
            throw new Error(response.status + " : " + response.url);
          };
        }
      } else {}

      var out = Module["print"] || console.log.bind(console);

      var err = Module["printErr"] || console.error.bind(console);

      // Merge back in the overrides
      Object.assign(Module, moduleOverrides);

      // Free the object hierarchy contained in the overrides, this lets the GC
      // reclaim data used.
      moduleOverrides = null;

      // Emit code to handle expected values on the Module object. This applies Module.x
      // to the proper local x. This has two benefits: first, we only emit it if it is
      // expected to arrive, and second, by using a local everywhere else that can be
      // minified.
      if (Module["arguments"]) arguments_ = Module["arguments"];

      if (Module["thisProgram"]) thisProgram = Module["thisProgram"];

      // perform assertions in shell.js after we set up out() and err(), as otherwise if an assertion fails it cannot print the message
      // end include: shell.js
      // include: preamble.js
      // === Preamble library stuff ===
      // Documentation for the public APIs defined in this file must be updated in:
      //    site/source/docs/api_reference/preamble.js.rst
      // A prebuilt local version of the documentation is available at:
      //    site/build/text/docs/api_reference/preamble.js.txt
      // You can also build docs locally as HTML or other formats in site/
      // An online HTML version (which may be of a different version of Emscripten)
      //    is up at http://kripken.github.io/emscripten-site/docs/api_reference/preamble.js.html
      var dynamicLibraries = Module["dynamicLibraries"] || [];

      var wasmBinary = Module["wasmBinary"];

      // Wasm globals
      var wasmMemory;

      //========================================
      // Runtime essentials
      //========================================
      // whether we are quitting the application. no code should run after this.
      // set in exit() and abort()
      var ABORT = false;

      // set by exit() and abort().  Passed to 'onExit' handler.
      // NOTE: This is also used as the process return code code in shell environments
      // but only when noExitRuntime is false.
      var EXITSTATUS;

      // In STRICT mode, we only define assert() when ASSERTIONS is set.  i.e. we
      // don't define it at all in release modes.  This matches the behaviour of
      // MINIMAL_RUNTIME.
      // TODO(sbc): Make this the default even without STRICT enabled.
      /** @type {function(*, string=)} */ function assert(condition, text) {
        if (!condition) {
          // This build was created without ASSERTIONS defined.  `assert()` should not
          // ever be called in this configuration but in case there are callers in
          // the wild leave this simple abort() implementation here for now.
          abort(text);
        }
      }

      // Memory management
      var HEAP,
        /** @type {!Int8Array} */ HEAP8,
        /** @type {!Uint8Array} */ HEAPU8,
        /** @type {!Int16Array} */ HEAP16,
        /** @type {!Uint16Array} */ HEAPU16,
        /** @type {!Int32Array} */ HEAP32,
        /** @type {!Uint32Array} */ HEAPU32,
        /** @type {!Float32Array} */ HEAPF32, /* BigInt64Array type is not correctly defined in closure
/** not-@type {!BigInt64Array} */
        HEAP64, /* BigUint64Array type is not correctly defined in closure
/** not-t@type {!BigUint64Array} */
        HEAPU64,
        /** @type {!Float64Array} */ HEAPF64;

      var HEAP_DATA_VIEW;

      var runtimeInitialized = false;

      /**
       * Indicates whether filename is delivered via file protocol (as opposed to http/https)
       * @noinline
       */ var isFileURI = (filename) => filename.startsWith("file://");

      // include: runtime_shared.js
      // include: runtime_stack_check.js
      // end include: runtime_stack_check.js
      // include: runtime_exceptions.js
      // end include: runtime_exceptions.js
      // include: runtime_debug.js
      // end include: runtime_debug.js
      // include: memoryprofiler.js
      // end include: memoryprofiler.js
      function updateMemoryViews() {
        var b = wasmMemory.buffer;
        Module["HEAP8"] = HEAP8 = new Int8Array(b);
        Module["HEAP16"] = HEAP16 = new Int16Array(b);
        Module["HEAPU8"] = HEAPU8 = new Uint8Array(b);
        Module["HEAPU16"] = HEAPU16 = new Uint16Array(b);
        Module["HEAP32"] = HEAP32 = new Int32Array(b);
        Module["HEAPU32"] = HEAPU32 = new Uint32Array(b);
        Module["HEAPF32"] = HEAPF32 = new Float32Array(b);
        Module["HEAPF64"] = HEAPF64 = new Float64Array(b);
        Module["HEAP64"] = HEAP64 = new BigInt64Array(b);
        Module["HEAPU64"] = HEAPU64 = new BigUint64Array(b);
        Module["HEAP_DATA_VIEW"] = HEAP_DATA_VIEW = new DataView(b);
        LE_HEAP_UPDATE();
      }

      // end include: runtime_shared.js
      // In non-standalone/normal mode, we create the memory here.
      // include: runtime_init_memory.js
      // Create the wasm memory. (Note: this only applies if IMPORTED_MEMORY is defined)
      // check for full engine support (use string 'subarray' to avoid closure compiler confusion)
      if (Module["wasmMemory"]) {
        wasmMemory = Module["wasmMemory"];
      } else {
        var INITIAL_MEMORY = Module["INITIAL_MEMORY"] || 33554432;
        /** @suppress {checkTypes} */ wasmMemory = new WebAssembly.Memory({
          "initial": INITIAL_MEMORY / 65536,
          // In theory we should not need to emit the maximum if we want "unlimited"
          // or 4GB of memory, but VMs error on that atm, see
          // https://github.com/emscripten-core/emscripten/issues/14130
          // And in the pthreads case we definitely need to emit a maximum. So
          // always emit one.
          "maximum": 32768,
        });
      }

      updateMemoryViews();

      // end include: runtime_init_memory.js
      var __RELOC_FUNCS__ = [];

      function preRun() {
        if (Module["preRun"]) {
          if (typeof Module["preRun"] == "function") {
            Module["preRun"] = [Module["preRun"]];
          }
          while (Module["preRun"].length) {
            addOnPreRun(Module["preRun"].shift());
          }
        }
        callRuntimeCallbacks(onPreRuns);
      }

      function initRuntime() {
        runtimeInitialized = true;
        callRuntimeCallbacks(__RELOC_FUNCS__);
        wasmExports["__wasm_call_ctors"]();
        callRuntimeCallbacks(onPostCtors);
      }

      function preMain() {}

      function postRun() {
        if (Module["postRun"]) {
          if (typeof Module["postRun"] == "function") {
            Module["postRun"] = [Module["postRun"]];
          }
          while (Module["postRun"].length) {
            addOnPostRun(Module["postRun"].shift());
          }
        }
        callRuntimeCallbacks(onPostRuns);
      }

      // A counter of dependencies for calling run(). If we need to
      // do asynchronous work before running, increment this and
      // decrement it. Incrementing must happen in a place like
      // Module.preRun (used by emcc to add file preloading).
      // Note that you can add dependencies in preRun, even though
      // it happens right before run - run will be postponed until
      // the dependencies are met.
      var runDependencies = 0;

      var dependenciesFulfilled = null;

      // overridden to take different actions when all run dependencies are fulfilled
      function getUniqueRunDependency(id) {
        return id;
      }

      function addRunDependency(id) {
        runDependencies++;
        Module["monitorRunDependencies"]?.(runDependencies);
      }

      function removeRunDependency(id) {
        runDependencies--;
        Module["monitorRunDependencies"]?.(runDependencies);
        if (runDependencies == 0) {
          if (dependenciesFulfilled) {
            var callback = dependenciesFulfilled;
            dependenciesFulfilled = null;
            callback();
          }
        }
      }

      /** @param {string|number=} what */ function abort(what) {
        Module["onAbort"]?.(what);
        what = "Aborted(" + what + ")";
        // TODO(sbc): Should we remove printing and leave it up to whoever
        // catches the exception?
        err(what);
        ABORT = true;
        what += ". Build with -sASSERTIONS for more info.";
        // Use a wasm runtime error, because a JS error might be seen as a foreign
        // exception, which means we'd run destructors on it. We need the error to
        // simply make the program stop.
        // FIXME This approach does not work in Wasm EH because it currently does not assume
        // all RuntimeErrors are from traps; it decides whether a RuntimeError is from
        // a trap or not based on a hidden field within the object. So at the moment
        // we don't have a way of throwing a wasm trap from JS. TODO Make a JS API that
        // allows this in the wasm spec.
        // Suppress closure compiler warning here. Closure compiler's builtin extern
        // definition for WebAssembly.RuntimeError claims it takes no arguments even
        // though it can.
        // TODO(https://github.com/google/closure-compiler/pull/3913): Remove if/when upstream closure gets fixed.
        /** @suppress {checkTypes} */ var e = new WebAssembly.RuntimeError(
          what,
        );
        readyPromiseReject(e);
        // Throw the error whether or not MODULARIZE is set because abort is used
        // in code paths apart from instantiation where an exception is expected
        // to be thrown when abort is called.
        throw e;
      }

      var wasmBinaryFile;

      function findWasmBinary() {
        if (Module["locateFile"]) {
          return locateFile("web-tree-sitter.wasm");
        }
        // Use bundler-friendly `new URL(..., import.meta.url)` pattern; works in browsers too.
        return new URL("web-tree-sitter.wasm", import.meta.url).href;
      }

      function getBinarySync(file) {
        if (file == wasmBinaryFile && wasmBinary) {
          return new Uint8Array(wasmBinary);
        }
        if (readBinary) {
          return readBinary(file);
        }
        throw "both async and sync fetching of the wasm failed";
      }

      async function getWasmBinary(binaryFile) {
        // If we don't have the binary yet, load it asynchronously using readAsync.
        if (!wasmBinary) {
          // Fetch the binary using readAsync
          try {
            var response = await readAsync(binaryFile);
            return new Uint8Array(response);
          } catch {}
        }
        // Otherwise, getBinarySync should be able to get it synchronously
        return getBinarySync(binaryFile);
      }

      async function instantiateArrayBuffer(binaryFile, imports) {
        try {
          var binary = await getWasmBinary(binaryFile);
          var instance = await WebAssembly.instantiate(binary, imports);
          return instance;
        } catch (reason) {
          err(`failed to asynchronously prepare wasm: ${reason}`);
          abort(reason);
        }
      }

      async function instantiateAsync(binary, binaryFile, imports) {
        if (
          !binary && typeof WebAssembly.instantiateStreaming == "function" &&
          !isFileURI(binaryFile) && !ENVIRONMENT_IS_NODE
        ) {
          try {
            var response = fetch(binaryFile, {
              credentials: "same-origin",
            });
            var instantiationResult = await WebAssembly.instantiateStreaming(
              response,
              imports,
            );
            return instantiationResult;
          } catch (reason) {
            // We expect the most common failure cause to be a bad MIME type for the binary,
            // in which case falling back to ArrayBuffer instantiation should work.
            err(`wasm streaming compile failed: ${reason}`);
            err("falling back to ArrayBuffer instantiation");
          }
        }
        return instantiateArrayBuffer(binaryFile, imports);
      }

      function getWasmImports() {
        // prepare imports
        return {
          "env": wasmImports,
          "wasi_snapshot_preview1": wasmImports,
          "GOT.mem": new Proxy(wasmImports, GOTHandler),
          "GOT.func": new Proxy(wasmImports, GOTHandler),
        };
      }

      // Create the wasm instance.
      // Receives the wasm imports, returns the exports.
      async function createWasm() {
        // Load the wasm module and create an instance of using native support in the JS engine.
        // handle a generated wasm instance, receiving its exports and
        // performing other necessary setup
        /** @param {WebAssembly.Module=} module*/ function receiveInstance(
          instance,
          module,
        ) {
          wasmExports = instance.exports;
          wasmExports = relocateExports(wasmExports, 1024);
          var metadata = getDylinkMetadata(module);
          if (metadata.neededDynlibs) {
            dynamicLibraries = metadata.neededDynlibs.concat(dynamicLibraries);
          }
          mergeLibSymbols(wasmExports, "main");
          LDSO.init();
          loadDylibs();
          __RELOC_FUNCS__.push(wasmExports["__wasm_apply_data_relocs"]);
          removeRunDependency("wasm-instantiate");
          return wasmExports;
        }
        // wait for the pthread pool (if any)
        addRunDependency("wasm-instantiate");
        // Prefer streaming instantiation if available.
        function receiveInstantiationResult(result) {
          // 'result' is a ResultObject object which has both the module and instance.
          // receiveInstance() will swap in the exports (to Module.asm) so they can be called
          return receiveInstance(result["instance"], result["module"]);
        }
        var info = getWasmImports();
        // User shell pages can write their own Module.instantiateWasm = function(imports, successCallback) callback
        // to manually instantiate the Wasm module themselves. This allows pages to
        // run the instantiation parallel to any other async startup actions they are
        // performing.
        // Also pthreads and wasm workers initialize the wasm instance through this
        // path.
        if (Module["instantiateWasm"]) {
          return new Promise((resolve, reject) => {
            Module["instantiateWasm"](info, (mod, inst) => {
              receiveInstance(mod, inst);
              resolve(mod.exports);
            });
          });
        }
        wasmBinaryFile ??= findWasmBinary();
        try {
          var result = await instantiateAsync(wasmBinary, wasmBinaryFile, info);
          var exports = receiveInstantiationResult(result);
          return exports;
        } catch (e) {
          // If instantiation fails, reject the module ready promise.
          readyPromiseReject(e);
          return Promise.reject(e);
        }
      }

      // end include: preamble.js
      // Begin JS library code
      class ExitStatus {
        name = "ExitStatus";
        constructor(status) {
          this.message = `Program terminated with exit(${status})`;
          this.status = status;
        }
      }

      var GOT = {};

      var currentModuleWeakSymbols = new Set([]);

      var GOTHandler = {
        get(obj, symName) {
          var rtn = GOT[symName];
          if (!rtn) {
            rtn = GOT[symName] = new WebAssembly.Global({
              "value": "i32",
              "mutable": true,
            });
          }
          if (!currentModuleWeakSymbols.has(symName)) {
            // Any non-weak reference to a symbol marks it as `required`, which
            // enabled `reportUndefinedSymbols` to report undefined symbol errors
            // correctly.
            rtn.required = true;
          }
          return rtn;
        },
      };

      var LE_ATOMICS_ADD = (heap, offset, value) => {
        const order = LE_ATOMICS_NATIVE_BYTE_ORDER[heap.BYTES_PER_ELEMENT - 1];
        const res = order(Atomics.add(heap, offset, order(value)));
        return heap.unsigned ? heap.unsigned(res) : res;
      };

      var LE_ATOMICS_AND = (heap, offset, value) => {
        const order = LE_ATOMICS_NATIVE_BYTE_ORDER[heap.BYTES_PER_ELEMENT - 1];
        const res = order(Atomics.and(heap, offset, order(value)));
        return heap.unsigned ? heap.unsigned(res) : res;
      };

      var LE_ATOMICS_COMPAREEXCHANGE = (
        heap,
        offset,
        expected,
        replacement,
      ) => {
        const order = LE_ATOMICS_NATIVE_BYTE_ORDER[heap.BYTES_PER_ELEMENT - 1];
        const res = order(
          Atomics.compareExchange(
            heap,
            offset,
            order(expected),
            order(replacement),
          ),
        );
        return heap.unsigned ? heap.unsigned(res) : res;
      };

      var LE_ATOMICS_EXCHANGE = (heap, offset, value) => {
        const order = LE_ATOMICS_NATIVE_BYTE_ORDER[heap.BYTES_PER_ELEMENT - 1];
        const res = order(Atomics.exchange(heap, offset, order(value)));
        return heap.unsigned ? heap.unsigned(res) : res;
      };

      var LE_ATOMICS_ISLOCKFREE = (size) => Atomics.isLockFree(size);

      var LE_ATOMICS_LOAD = (heap, offset) => {
        const order = LE_ATOMICS_NATIVE_BYTE_ORDER[heap.BYTES_PER_ELEMENT - 1];
        const res = order(Atomics.load(heap, offset));
        return heap.unsigned ? heap.unsigned(res) : res;
      };

      var LE_ATOMICS_NATIVE_BYTE_ORDER = [];

      var LE_ATOMICS_NOTIFY = (heap, offset, count) =>
        Atomics.notify(heap, offset, count);

      var LE_ATOMICS_OR = (heap, offset, value) => {
        const order = LE_ATOMICS_NATIVE_BYTE_ORDER[heap.BYTES_PER_ELEMENT - 1];
        const res = order(Atomics.or(heap, offset, order(value)));
        return heap.unsigned ? heap.unsigned(res) : res;
      };

      var LE_ATOMICS_STORE = (heap, offset, value) => {
        const order = LE_ATOMICS_NATIVE_BYTE_ORDER[heap.BYTES_PER_ELEMENT - 1];
        Atomics.store(heap, offset, order(value));
      };

      var LE_ATOMICS_SUB = (heap, offset, value) => {
        const order = LE_ATOMICS_NATIVE_BYTE_ORDER[heap.BYTES_PER_ELEMENT - 1];
        const res = order(Atomics.sub(heap, offset, order(value)));
        return heap.unsigned ? heap.unsigned(res) : res;
      };

      var LE_ATOMICS_WAIT = (heap, offset, value, timeout) => {
        const order = LE_ATOMICS_NATIVE_BYTE_ORDER[heap.BYTES_PER_ELEMENT - 1];
        return Atomics.wait(heap, offset, order(value), timeout);
      };

      var LE_ATOMICS_WAITASYNC = (heap, offset, value, timeout) => {
        const order = LE_ATOMICS_NATIVE_BYTE_ORDER[heap.BYTES_PER_ELEMENT - 1];
        return Atomics.waitAsync(heap, offset, order(value), timeout);
      };

      var LE_ATOMICS_XOR = (heap, offset, value) => {
        const order = LE_ATOMICS_NATIVE_BYTE_ORDER[heap.BYTES_PER_ELEMENT - 1];
        const res = order(Atomics.xor(heap, offset, order(value)));
        return heap.unsigned ? heap.unsigned(res) : res;
      };

      var LE_HEAP_LOAD_F32 = (byteOffset) =>
        HEAP_DATA_VIEW.getFloat32(byteOffset, true);

      var LE_HEAP_LOAD_F64 = (byteOffset) =>
        HEAP_DATA_VIEW.getFloat64(byteOffset, true);

      var LE_HEAP_LOAD_I16 = (byteOffset) =>
        HEAP_DATA_VIEW.getInt16(byteOffset, true);

      var LE_HEAP_LOAD_I32 = (byteOffset) =>
        HEAP_DATA_VIEW.getInt32(byteOffset, true);

      var LE_HEAP_LOAD_U16 = (byteOffset) =>
        HEAP_DATA_VIEW.getUint16(byteOffset, true);

      var LE_HEAP_LOAD_U32 = (byteOffset) =>
        HEAP_DATA_VIEW.getUint32(byteOffset, true);

      var LE_HEAP_STORE_F32 = (byteOffset, value) =>
        HEAP_DATA_VIEW.setFloat32(byteOffset, value, true);

      var LE_HEAP_STORE_F64 = (byteOffset, value) =>
        HEAP_DATA_VIEW.setFloat64(byteOffset, value, true);

      var LE_HEAP_STORE_I16 = (byteOffset, value) =>
        HEAP_DATA_VIEW.setInt16(byteOffset, value, true);

      var LE_HEAP_STORE_I32 = (byteOffset, value) =>
        HEAP_DATA_VIEW.setInt32(byteOffset, value, true);

      var LE_HEAP_STORE_U16 = (byteOffset, value) =>
        HEAP_DATA_VIEW.setUint16(byteOffset, value, true);

      var LE_HEAP_STORE_U32 = (byteOffset, value) =>
        HEAP_DATA_VIEW.setUint32(byteOffset, value, true);

      var callRuntimeCallbacks = (callbacks) => {
        while (callbacks.length > 0) {
          // Pass the module as the first argument.
          callbacks.shift()(Module);
        }
      };

      var onPostRuns = [];

      var addOnPostRun = (cb) => onPostRuns.unshift(cb);

      var onPreRuns = [];

      var addOnPreRun = (cb) => onPreRuns.unshift(cb);

      var UTF8Decoder = typeof TextDecoder != "undefined"
        ? new TextDecoder()
        : undefined;

      /**
       * Given a pointer 'idx' to a null-terminated UTF8-encoded string in the given
       * array that contains uint8 values, returns a copy of that string as a
       * Javascript String object.
       * heapOrArray is either a regular array, or a JavaScript typed array view.
       * @param {number=} idx
       * @param {number=} maxBytesToRead
       * @return {string}
       */ var UTF8ArrayToString = (
        heapOrArray,
        idx = 0,
        maxBytesToRead = NaN,
      ) => {
        var endIdx = idx + maxBytesToRead;
        var endPtr = idx;
        // TextDecoder needs to know the byte length in advance, it doesn't stop on
        // null terminator by itself.  Also, use the length info to avoid running tiny
        // strings through TextDecoder, since .subarray() allocates garbage.
        // (As a tiny code save trick, compare endPtr against endIdx using a negation,
        // so that undefined/NaN means Infinity)
        while (heapOrArray[endPtr] && !(endPtr >= endIdx)) ++endPtr;
        if (endPtr - idx > 16 && heapOrArray.buffer && UTF8Decoder) {
          return UTF8Decoder.decode(heapOrArray.subarray(idx, endPtr));
        }
        var str = "";
        // If building with TextDecoder, we have already computed the string length
        // above, so test loop end condition against that
        while (idx < endPtr) {
          // For UTF8 byte structure, see:
          // http://en.wikipedia.org/wiki/UTF-8#Description
          // https://www.ietf.org/rfc/rfc2279.txt
          // https://tools.ietf.org/html/rfc3629
          var u0 = heapOrArray[idx++];
          if (!(u0 & 128)) {
            str += String.fromCharCode(u0);
            continue;
          }
          var u1 = heapOrArray[idx++] & 63;
          if ((u0 & 224) == 192) {
            str += String.fromCharCode(((u0 & 31) << 6) | u1);
            continue;
          }
          var u2 = heapOrArray[idx++] & 63;
          if ((u0 & 240) == 224) {
            u0 = ((u0 & 15) << 12) | (u1 << 6) | u2;
          } else {
            u0 = ((u0 & 7) << 18) | (u1 << 12) | (u2 << 6) |
              (heapOrArray[idx++] & 63);
          }
          if (u0 < 65536) {
            str += String.fromCharCode(u0);
          } else {
            var ch = u0 - 65536;
            str += String.fromCharCode(55296 | (ch >> 10), 56320 | (ch & 1023));
          }
        }
        return str;
      };

      var getDylinkMetadata = (binary) => {
        var offset = 0;
        var end = 0;
        function getU8() {
          return binary[offset++];
        }
        function getLEB() {
          var ret = 0;
          var mul = 1;
          while (1) {
            var byte = binary[offset++];
            ret += (byte & 127) * mul;
            mul *= 128;
            if (!(byte & 128)) break;
          }
          return ret;
        }
        function getString() {
          var len = getLEB();
          offset += len;
          return UTF8ArrayToString(binary, offset - len, len);
        }
        function getStringList() {
          var count = getLEB();
          var rtn = [];
          while (count--) rtn.push(getString());
          return rtn;
        }
        /** @param {string=} message */ function failIf(condition, message) {
          if (condition) throw new Error(message);
        }
        if (binary instanceof WebAssembly.Module) {
          var dylinkSection = WebAssembly.Module.customSections(
            binary,
            "dylink.0",
          );
          failIf(dylinkSection.length === 0, "need dylink section");
          binary = new Uint8Array(dylinkSection[0]);
          end = binary.length;
        } else {
          var int32View = new Uint32Array(
            new Uint8Array(binary.subarray(0, 24)).buffer,
          );
          var magicNumberFound = int32View[0] == 1836278016 ||
            int32View[0] == 6386541;
          failIf(!magicNumberFound, "need to see wasm magic number");
          // \0asm
          // we should see the dylink custom section right after the magic number and wasm version
          failIf(binary[8] !== 0, "need the dylink section to be first");
          offset = 9;
          var section_size = getLEB();
          //section size
          end = offset + section_size;
          var name = getString();
          failIf(name !== "dylink.0");
        }
        var customSection = {
          neededDynlibs: [],
          tlsExports: new Set(),
          weakImports: new Set(),
          runtimePaths: [],
        };
        var WASM_DYLINK_MEM_INFO = 1;
        var WASM_DYLINK_NEEDED = 2;
        var WASM_DYLINK_EXPORT_INFO = 3;
        var WASM_DYLINK_IMPORT_INFO = 4;
        var WASM_DYLINK_RUNTIME_PATH = 5;
        var WASM_SYMBOL_TLS = 256;
        var WASM_SYMBOL_BINDING_MASK = 3;
        var WASM_SYMBOL_BINDING_WEAK = 1;
        while (offset < end) {
          var subsectionType = getU8();
          var subsectionSize = getLEB();
          if (subsectionType === WASM_DYLINK_MEM_INFO) {
            customSection.memorySize = getLEB();
            customSection.memoryAlign = getLEB();
            customSection.tableSize = getLEB();
            customSection.tableAlign = getLEB();
          } else if (subsectionType === WASM_DYLINK_NEEDED) {
            customSection.neededDynlibs = getStringList();
          } else if (subsectionType === WASM_DYLINK_EXPORT_INFO) {
            var count = getLEB();
            while (count--) {
              var symname = getString();
              var flags = getLEB();
              if (flags & WASM_SYMBOL_TLS) {
                customSection.tlsExports.add(symname);
              }
            }
          } else if (subsectionType === WASM_DYLINK_IMPORT_INFO) {
            var count = getLEB();
            while (count--) {
              var modname = getString();
              var symname = getString();
              var flags = getLEB();
              if (
                (flags & WASM_SYMBOL_BINDING_MASK) == WASM_SYMBOL_BINDING_WEAK
              ) {
                customSection.weakImports.add(symname);
              }
            }
          } else if (subsectionType === WASM_DYLINK_RUNTIME_PATH) {
            customSection.runtimePaths = getStringList();
          } else {
            // unknown subsection
            offset += subsectionSize;
          }
        }
        return customSection;
      };

      /**
       * @param {number} ptr
       * @param {string} type
       */ function getValue(ptr, type = "i8") {
        if (type.endsWith("*")) type = "*";
        switch (type) {
          case "i1":
            return HEAP8[ptr];

          case "i8":
            return HEAP8[ptr];

          case "i16":
            return LE_HEAP_LOAD_I16((ptr >> 1) * 2);

          case "i32":
            return LE_HEAP_LOAD_I32((ptr >> 2) * 4);

          case "i64":
            return HEAP64[ptr >> 3];

          case "float":
            return LE_HEAP_LOAD_F32((ptr >> 2) * 4);

          case "double":
            return LE_HEAP_LOAD_F64((ptr >> 3) * 8);

          case "*":
            return LE_HEAP_LOAD_U32((ptr >> 2) * 4);

          default:
            abort(`invalid type for getValue: ${type}`);
        }
      }

      var newDSO = (name, handle, syms) => {
        var dso = {
          refcount: Infinity,
          name,
          exports: syms,
          global: true,
        };
        LDSO.loadedLibsByName[name] = dso;
        if (handle != undefined) {
          LDSO.loadedLibsByHandle[handle] = dso;
        }
        return dso;
      };

      var LDSO = {
        loadedLibsByName: {},
        loadedLibsByHandle: {},
        init() {
          newDSO("__main__", 0, wasmImports);
        },
      };

      var ___heap_base = 78224;

      var alignMemory = (size, alignment) =>
        Math.ceil(size / alignment) * alignment;

      var getMemory = (size) => {
        // After the runtime is initialized, we must only use sbrk() normally.
        if (runtimeInitialized) {
          // Currently we don't support freeing of static data when modules are
          // unloaded via dlclose.  This function is tagged as `noleakcheck` to
          // avoid having this reported as leak.
          return _calloc(size, 1);
        }
        var ret = ___heap_base;
        // Keep __heap_base stack aligned.
        var end = ret + alignMemory(size, 16);
        ___heap_base = end;
        GOT["__heap_base"].value = end;
        return ret;
      };

      var isInternalSym = (symName) =>
        [
          "__cpp_exception",
          "__c_longjmp",
          "__wasm_apply_data_relocs",
          "__dso_handle",
          "__tls_size",
          "__tls_align",
          "__set_stack_limits",
          "_emscripten_tls_init",
          "__wasm_init_tls",
          "__wasm_call_ctors",
          "__start_em_asm",
          "__stop_em_asm",
          "__start_em_js",
          "__stop_em_js",
        ].includes(symName) || symName.startsWith("__em_js__");

      var uleb128Encode = (n, target) => {
        if (n < 128) {
          target.push(n);
        } else {
          target.push((n % 128) | 128, n >> 7);
        }
      };

      var sigToWasmTypes = (sig) => {
        var typeNames = {
          "i": "i32",
          "j": "i64",
          "f": "f32",
          "d": "f64",
          "e": "externref",
          "p": "i32",
        };
        var type = {
          parameters: [],
          results: sig[0] == "v" ? [] : [typeNames[sig[0]]],
        };
        for (var i = 1; i < sig.length; ++i) {
          type.parameters.push(typeNames[sig[i]]);
        }
        return type;
      };

      var generateFuncType = (sig, target) => {
        var sigRet = sig.slice(0, 1);
        var sigParam = sig.slice(1);
        var typeCodes = {
          "i": 127,
          // i32
          "p": 127,
          // i32
          "j": 126,
          // i64
          "f": 125,
          // f32
          "d": 124,
          // f64
          "e": 111,
        };
        // Parameters, length + signatures
        target.push(96);
        uleb128Encode(sigParam.length, target);
        for (var paramType of sigParam) {
          target.push(typeCodes[paramType]);
        }
        // Return values, length + signatures
        // With no multi-return in MVP, either 0 (void) or 1 (anything else)
        if (sigRet == "v") {
          target.push(0);
        } else {
          target.push(1, typeCodes[sigRet]);
        }
      };

      var convertJsFunctionToWasm = (func, sig) => {
        // If the type reflection proposal is available, use the new
        // "WebAssembly.Function" constructor.
        // Otherwise, construct a minimal wasm module importing the JS function and
        // re-exporting it.
        if (typeof WebAssembly.Function == "function") {
          return new WebAssembly.Function(sigToWasmTypes(sig), func);
        }
        // The module is static, with the exception of the type section, which is
        // generated based on the signature passed in.
        var typeSectionBody = [1];
        generateFuncType(sig, typeSectionBody);
        // Rest of the module is static
        var bytes = [
          0,
          97,
          115,
          109, // magic ("\0asm")
          1,
          0,
          0,
          0, // version: 1
          1,
        ];
        // Write the overall length of the type section followed by the body
        uleb128Encode(typeSectionBody.length, bytes);
        bytes.push(...typeSectionBody);
        // The rest of the module is static
        bytes.push(
          2,
          7, // import section
          // (import "e" "f" (func 0 (type 0)))
          1,
          1,
          101,
          1,
          102,
          0,
          0,
          7,
          5, // export section
          // (export "f" (func 0 (type 0)))
          1,
          1,
          102,
          0,
          0,
        );
        // We can compile this wasm module synchronously because it is very small.
        // This accepts an import (at "e.f"), that it reroutes to an export (at "f")
        var module = new WebAssembly.Module(new Uint8Array(bytes));
        var instance = new WebAssembly.Instance(module, {
          "e": {
            "f": func,
          },
        });
        var wrappedFunc = instance.exports["f"];
        return wrappedFunc;
      };

      var wasmTableMirror = [];

      /** @type {WebAssembly.Table} */ var wasmTable = new WebAssembly.Table({
        "initial": 31,
        "element": "anyfunc",
      });

      var getWasmTableEntry = (funcPtr) => {
        var func = wasmTableMirror[funcPtr];
        if (!func) {
          /** @suppress {checkTypes} */ wasmTableMirror[funcPtr] =
            func =
              wasmTable.get(funcPtr);
        }
        return func;
      };

      var updateTableMap = (offset, count) => {
        if (functionsInTableMap) {
          for (var i = offset; i < offset + count; i++) {
            var item = getWasmTableEntry(i);
            // Ignore null values.
            if (item) {
              functionsInTableMap.set(item, i);
            }
          }
        }
      };

      var functionsInTableMap;

      var getFunctionAddress = (func) => {
        // First, create the map if this is the first use.
        if (!functionsInTableMap) {
          functionsInTableMap = new WeakMap();
          updateTableMap(0, wasmTable.length);
        }
        return functionsInTableMap.get(func) || 0;
      };

      var freeTableIndexes = [];

      var getEmptyTableSlot = () => {
        // Reuse a free index if there is one, otherwise grow.
        if (freeTableIndexes.length) {
          return freeTableIndexes.pop();
        }
        // Grow the table
        try {
          /** @suppress {checkTypes} */ wasmTable.grow(1);
        } catch (err) {
          if (!(err instanceof RangeError)) {
            throw err;
          }
          throw "Unable to grow wasm table. Set ALLOW_TABLE_GROWTH.";
        }
        return wasmTable.length - 1;
      };

      var setWasmTableEntry = (idx, func) => {
        /** @suppress {checkTypes} */ wasmTable.set(idx, func);
        // With ABORT_ON_WASM_EXCEPTIONS wasmTable.get is overridden to return wrapped
        // functions so we need to call it here to retrieve the potential wrapper correctly
        // instead of just storing 'func' directly into wasmTableMirror
        /** @suppress {checkTypes} */ wasmTableMirror[idx] = wasmTable.get(idx);
      };

      /** @param {string=} sig */ var addFunction = (func, sig) => {
        // Check if the function is already in the table, to ensure each function
        // gets a unique index.
        var rtn = getFunctionAddress(func);
        if (rtn) {
          return rtn;
        }
        // It's not in the table, add it now.
        var ret = getEmptyTableSlot();
        // Set the new value.
        try {
          // Attempting to call this with JS function will cause of table.set() to fail
          setWasmTableEntry(ret, func);
        } catch (err) {
          if (!(err instanceof TypeError)) {
            throw err;
          }
          var wrapped = convertJsFunctionToWasm(func, sig);
          setWasmTableEntry(ret, wrapped);
        }
        functionsInTableMap.set(func, ret);
        return ret;
      };

      var updateGOT = (exports, replace) => {
        for (var symName in exports) {
          if (isInternalSym(symName)) {
            continue;
          }
          var value = exports[symName];
          GOT[symName] ||= new WebAssembly.Global({
            "value": "i32",
            "mutable": true,
          });
          if (replace || GOT[symName].value == 0) {
            if (typeof value == "function") {
              GOT[symName].value = addFunction(value);
            } else if (typeof value == "number") {
              GOT[symName].value = value;
            } else {
              err(`unhandled export type for '${symName}': ${typeof value}`);
            }
          }
        }
      };

      /** @param {boolean=} replace */ var relocateExports = (
        exports,
        memoryBase,
        replace,
      ) => {
        var relocated = {};
        for (var e in exports) {
          var value = exports[e];
          if (typeof value == "object") {
            // a breaking change in the wasm spec, globals are now objects
            // https://github.com/WebAssembly/mutable-global/issues/1
            value = value.value;
          }
          if (typeof value == "number") {
            value += memoryBase;
          }
          relocated[e] = value;
        }
        updateGOT(relocated, replace);
        return relocated;
      };

      var isSymbolDefined = (symName) => {
        // Ignore 'stub' symbols that are auto-generated as part of the original
        // `wasmImports` used to instantiate the main module.
        var existing = wasmImports[symName];
        if (!existing || existing.stub) {
          return false;
        }
        return true;
      };

      var dynCall = (sig, ptr, args = []) => {
        var rtn = getWasmTableEntry(ptr)(...args);
        return rtn;
      };

      var stackSave = () => _emscripten_stack_get_current();

      var stackRestore = (val) => __emscripten_stack_restore(val);

      var createInvokeFunction = (sig) => (ptr, ...args) => {
        var sp = stackSave();
        try {
          return dynCall(sig, ptr, args);
        } catch (e) {
          stackRestore(sp);
          // Create a try-catch guard that rethrows the Emscripten EH exception.
          // Exceptions thrown from C++ will be a pointer (number) and longjmp
          // will throw the number Infinity. Use the compact and fast "e !== e+0"
          // test to check if e was not a Number.
          if (e !== e + 0) throw e;
          _setThrew(1, 0);
          // In theory this if statement could be done on
          // creating the function, but I just added this to
          // save wasting code space as it only happens on exception.
          if (sig[0] == "j") return 0n;
        }
      };

      var resolveGlobalSymbol = (symName, direct = false) => {
        var sym;
        if (isSymbolDefined(symName)) {
          sym = wasmImports[symName];
        } else if (symName.startsWith("invoke_")) {
          // Create (and cache) new invoke_ functions on demand.
          sym = wasmImports[symName] = createInvokeFunction(
            symName.split("_")[1],
          );
        }
        return {
          sym,
          name: symName,
        };
      };

      var onPostCtors = [];

      var addOnPostCtor = (cb) => onPostCtors.unshift(cb);

      /**
       * Given a pointer 'ptr' to a null-terminated UTF8-encoded string in the
       * emscripten HEAP, returns a copy of that string as a Javascript String object.
       *
       * @param {number} ptr
       * @param {number=} maxBytesToRead - An optional length that specifies the
       *   maximum number of bytes to read. You can omit this parameter to scan the
       *   string until the first 0 byte. If maxBytesToRead is passed, and the string
       *   at [ptr, ptr+maxBytesToReadr[ contains a null byte in the middle, then the
       *   string will cut short at that byte index (i.e. maxBytesToRead will not
       *   produce a string of exact length [ptr, ptr+maxBytesToRead[) N.B. mixing
       *   frequent uses of UTF8ToString() with and without maxBytesToRead may throw
       *   JS JIT optimizations off, so it is worth to consider consistently using one
       * @return {string}
       */ var UTF8ToString = (ptr, maxBytesToRead) =>
        ptr ? UTF8ArrayToString(HEAPU8, ptr, maxBytesToRead) : "";

      /**
       * @param {string=} libName
       * @param {Object=} localScope
       * @param {number=} handle
       */ var loadWebAssemblyModule = (
        binary,
        flags,
        libName,
        localScope,
        handle,
      ) => {
        var metadata = getDylinkMetadata(binary);
        currentModuleWeakSymbols = metadata.weakImports;
        // loadModule loads the wasm module after all its dependencies have been loaded.
        // can be called both sync/async.
        function loadModule() {
          // alignments are powers of 2
          var memAlign = Math.pow(2, metadata.memoryAlign);
          // prepare memory
          var memoryBase = metadata.memorySize
            ? alignMemory(getMemory(metadata.memorySize + memAlign), memAlign)
            : 0;
          // TODO: add to cleanups
          var tableBase = metadata.tableSize ? wasmTable.length : 0;
          if (handle) {
            HEAP8[handle + (8)] = 1;
            LE_HEAP_STORE_U32(((handle + (12)) >> 2) * 4, memoryBase);
            LE_HEAP_STORE_I32(((handle + (16)) >> 2) * 4, metadata.memorySize);
            LE_HEAP_STORE_U32(((handle + (20)) >> 2) * 4, tableBase);
            LE_HEAP_STORE_I32(((handle + (24)) >> 2) * 4, metadata.tableSize);
          }
          if (metadata.tableSize) {
            wasmTable.grow(metadata.tableSize);
          }
          // This is the export map that we ultimately return.  We declare it here
          // so it can be used within resolveSymbol.  We resolve symbols against
          // this local symbol map in the case there they are not present on the
          // global Module object.  We need this fallback because Modules sometime
          // need to import their own symbols
          var moduleExports;
          function resolveSymbol(sym) {
            var resolved = resolveGlobalSymbol(sym).sym;
            if (!resolved && localScope) {
              resolved = localScope[sym];
            }
            if (!resolved) {
              resolved = moduleExports[sym];
            }
            return resolved;
          }
          // TODO kill ↓↓↓ (except "symbols local to this module", it will likely be
          // not needed if we require that if A wants symbols from B it has to link
          // to B explicitly: similarly to -Wl,--no-undefined)
          // wasm dynamic libraries are pure wasm, so they cannot assist in
          // their own loading. When side module A wants to import something
          // provided by a side module B that is loaded later, we need to
          // add a layer of indirection, but worse, we can't even tell what
          // to add the indirection for, without inspecting what A's imports
          // are. To do that here, we use a JS proxy (another option would
          // be to inspect the binary directly).
          var proxyHandler = {
            get(stubs, prop) {
              // symbols that should be local to this module
              switch (prop) {
                case "__memory_base":
                  return memoryBase;

                case "__table_base":
                  return tableBase;
              }
              if (prop in wasmImports && !wasmImports[prop].stub) {
                // No stub needed, symbol already exists in symbol table
                var res = wasmImports[prop];
                return res;
              }
              // Return a stub function that will resolve the symbol
              // when first called.
              if (!(prop in stubs)) {
                var resolved;
                stubs[prop] = (...args) => {
                  resolved ||= resolveSymbol(prop);
                  return resolved(...args);
                };
              }
              return stubs[prop];
            },
          };
          var proxy = new Proxy({}, proxyHandler);
          var info = {
            "GOT.mem": new Proxy({}, GOTHandler),
            "GOT.func": new Proxy({}, GOTHandler),
            "env": proxy,
            "wasi_snapshot_preview1": proxy,
          };
          function postInstantiation(module, instance) {
            // add new entries to functionsInTableMap
            updateTableMap(tableBase, metadata.tableSize);
            moduleExports = relocateExports(instance.exports, memoryBase);
            if (!flags.allowUndefined) {
              reportUndefinedSymbols();
            }
            function addEmAsm(addr, body) {
              var args = [];
              var arity = 0;
              for (; arity < 16; arity++) {
                if (body.indexOf("$" + arity) != -1) {
                  args.push("$" + arity);
                } else {
                  break;
                }
              }
              args = args.join(",");
              var func = `(${args}) => { ${body} };`;
              ASM_CONSTS[start] = eval(func);
            }
            // Add any EM_ASM function that exist in the side module
            if ("__start_em_asm" in moduleExports) {
              var start = moduleExports["__start_em_asm"];
              var stop = moduleExports["__stop_em_asm"];
              while (start < stop) {
                var jsString = UTF8ToString(start);
                addEmAsm(start, jsString);
                start = HEAPU8.indexOf(0, start) + 1;
              }
            }
            function addEmJs(name, cSig, body) {
              // The signature here is a C signature (e.g. "(int foo, char* bar)").
              // See `create_em_js` in emcc.py` for the build-time version of this
              // code.
              var jsArgs = [];
              cSig = cSig.slice(1, -1);
              if (cSig != "void") {
                cSig = cSig.split(",");
                for (var i in cSig) {
                  var jsArg = cSig[i].split(" ").pop();
                  jsArgs.push(jsArg.replace("*", ""));
                }
              }
              var func = `(${jsArgs}) => ${body};`;
              moduleExports[name] = eval(func);
            }
            for (var name in moduleExports) {
              if (name.startsWith("__em_js__")) {
                var start = moduleExports[name];
                var jsString = UTF8ToString(start);
                // EM_JS strings are stored in the data section in the form
                // SIG<::>BODY.
                var parts = jsString.split("<::>");
                addEmJs(name.replace("__em_js__", ""), parts[0], parts[1]);
                delete moduleExports[name];
              }
            }
            // initialize the module
            var applyRelocs = moduleExports["__wasm_apply_data_relocs"];
            if (applyRelocs) {
              if (runtimeInitialized) {
                applyRelocs();
              } else {
                __RELOC_FUNCS__.push(applyRelocs);
              }
            }
            var init = moduleExports["__wasm_call_ctors"];
            if (init) {
              if (runtimeInitialized) {
                init();
              } else {
                // we aren't ready to run compiled code yet
                addOnPostCtor(init);
              }
            }
            return moduleExports;
          }
          if (flags.loadAsync) {
            if (binary instanceof WebAssembly.Module) {
              var instance = new WebAssembly.Instance(binary, info);
              return Promise.resolve(postInstantiation(binary, instance));
            }
            return WebAssembly.instantiate(binary, info).then((result) =>
              postInstantiation(result.module, result.instance)
            );
          }
          var module = binary instanceof WebAssembly.Module
            ? binary
            : new WebAssembly.Module(binary);
          var instance = new WebAssembly.Instance(module, info);
          return postInstantiation(module, instance);
        }
        // now load needed libraries and the module itself.
        if (flags.loadAsync) {
          return metadata.neededDynlibs.reduce(
            (chain, dynNeeded) =>
              chain.then(() =>
                loadDynamicLibrary(dynNeeded, flags, localScope)
              ),
            Promise.resolve(),
          ).then(loadModule);
        }
        metadata.neededDynlibs.forEach((needed) =>
          loadDynamicLibrary(needed, flags, localScope)
        );
        return loadModule();
      };

      var mergeLibSymbols = (exports, libName) => {
        // add symbols into global namespace TODO: weak linking etc.
        for (var [sym, exp] of Object.entries(exports)) {
          // When RTLD_GLOBAL is enabled, the symbols defined by this shared object
          // will be made available for symbol resolution of subsequently loaded
          // shared objects.
          // We should copy the symbols (which include methods and variables) from
          // SIDE_MODULE to MAIN_MODULE.
          const setImport = (target) => {
            if (!isSymbolDefined(target)) {
              wasmImports[target] = exp;
            }
          };
          setImport(sym);
          // Special case for handling of main symbol:  If a side module exports
          // `main` that also acts a definition for `__main_argc_argv` and vice
          // versa.
          const main_alias = "__main_argc_argv";
          if (sym == "main") {
            setImport(main_alias);
          }
          if (sym == main_alias) {
            setImport("main");
          }
        }
      };

      var asyncLoad = async (url) => {
        var arrayBuffer = await readAsync(url);
        return new Uint8Array(arrayBuffer);
      };

      /**
       * @param {number=} handle
       * @param {Object=} localScope
       */ function loadDynamicLibrary(
        libName,
        flags = {
          global: true,
          nodelete: true,
        },
        localScope,
        handle,
      ) {
        // when loadDynamicLibrary did not have flags, libraries were loaded
        // globally & permanently
        var dso = LDSO.loadedLibsByName[libName];
        if (dso) {
          // the library is being loaded or has been loaded already.
          if (!flags.global) {
            if (localScope) {
              Object.assign(localScope, dso.exports);
            }
          } else if (!dso.global) {
            // The library was previously loaded only locally but not
            // we have a request with global=true.
            dso.global = true;
            mergeLibSymbols(dso.exports, libName);
          }
          // same for "nodelete"
          if (flags.nodelete && dso.refcount !== Infinity) {
            dso.refcount = Infinity;
          }
          dso.refcount++;
          if (handle) {
            LDSO.loadedLibsByHandle[handle] = dso;
          }
          return flags.loadAsync ? Promise.resolve(true) : true;
        }
        // allocate new DSO
        dso = newDSO(libName, handle, "loading");
        dso.refcount = flags.nodelete ? Infinity : 1;
        dso.global = flags.global;
        // libName -> libData
        function loadLibData() {
          // for wasm, we can use fetch for async, but for fs mode we can only imitate it
          if (handle) {
            var data = LE_HEAP_LOAD_U32(((handle + (28)) >> 2) * 4);
            var dataSize = LE_HEAP_LOAD_U32(((handle + (32)) >> 2) * 4);
            if (data && dataSize) {
              var libData = HEAP8.slice(data, data + dataSize);
              return flags.loadAsync ? Promise.resolve(libData) : libData;
            }
          }
          var libFile = locateFile(libName);
          if (flags.loadAsync) {
            return asyncLoad(libFile);
          }
          // load the binary synchronously
          if (!readBinary) {
            throw new Error(
              `${libFile}: file not found, and synchronous loading of external files is not available`,
            );
          }
          return readBinary(libFile);
        }
        // libName -> exports
        function getExports() {
          // module not preloaded - load lib data and create new module from it
          if (flags.loadAsync) {
            return loadLibData().then((libData) =>
              loadWebAssemblyModule(libData, flags, libName, localScope, handle)
            );
          }
          return loadWebAssemblyModule(
            loadLibData(),
            flags,
            libName,
            localScope,
            handle,
          );
        }
        // module for lib is loaded - update the dso & global namespace
        function moduleLoaded(exports) {
          if (dso.global) {
            mergeLibSymbols(exports, libName);
          } else if (localScope) {
            Object.assign(localScope, exports);
          }
          dso.exports = exports;
        }
        if (flags.loadAsync) {
          return getExports().then((exports) => {
            moduleLoaded(exports);
            return true;
          });
        }
        moduleLoaded(getExports());
        return true;
      }

      var reportUndefinedSymbols = () => {
        for (var [symName, entry] of Object.entries(GOT)) {
          if (entry.value == 0) {
            var value = resolveGlobalSymbol(symName, true).sym;
            if (!value && !entry.required) {
              // Ignore undefined symbols that are imported as weak.
              continue;
            }
            if (typeof value == "function") {
              /** @suppress {checkTypes} */ entry.value = addFunction(
                value,
                value.sig,
              );
            } else if (typeof value == "number") {
              entry.value = value;
            } else {
              throw new Error(
                `bad export type for '${symName}': ${typeof value}`,
              );
            }
          }
        }
      };

      var loadDylibs = () => {
        if (!dynamicLibraries.length) {
          reportUndefinedSymbols();
          return;
        }
        // Load binaries asynchronously
        addRunDependency("loadDylibs");
        dynamicLibraries.reduce(
          (chain, lib) =>
            chain.then(() =>
              loadDynamicLibrary(lib, {
                loadAsync: true,
                global: true,
                nodelete: true,
                allowUndefined: true,
              })
            ),
          Promise.resolve(),
        ).then(() => {
          // we got them all, wonderful
          reportUndefinedSymbols();
          removeRunDependency("loadDylibs");
        });
      };

      var noExitRuntime = Module["noExitRuntime"] || true;

      /**
       * @param {number} ptr
       * @param {number} value
       * @param {string} type
       */ function setValue(ptr, value, type = "i8") {
        if (type.endsWith("*")) type = "*";
        switch (type) {
          case "i1":
            HEAP8[ptr] = value;
            break;

          case "i8":
            HEAP8[ptr] = value;
            break;

          case "i16":
            LE_HEAP_STORE_I16((ptr >> 1) * 2, value);
            break;

          case "i32":
            LE_HEAP_STORE_I32((ptr >> 2) * 4, value);
            break;

          case "i64":
            HEAP64[ptr >> 3] = BigInt(value);
            break;

          case "float":
            LE_HEAP_STORE_F32((ptr >> 2) * 4, value);
            break;

          case "double":
            LE_HEAP_STORE_F64((ptr >> 3) * 8, value);
            break;

          case "*":
            LE_HEAP_STORE_U32((ptr >> 2) * 4, value);
            break;

          default:
            abort(`invalid type for setValue: ${type}`);
        }
      }

      var ___memory_base = new WebAssembly.Global({
        "value": "i32",
        "mutable": false,
      }, 1024);

      var ___stack_pointer = new WebAssembly.Global({
        "value": "i32",
        "mutable": true,
      }, 78224);

      var ___table_base = new WebAssembly.Global({
        "value": "i32",
        "mutable": false,
      }, 1);

      var __abort_js = () => abort("");

      __abort_js.sig = "v";

      var _emscripten_get_now = () => performance.now();

      _emscripten_get_now.sig = "d";

      var _emscripten_date_now = () => Date.now();

      _emscripten_date_now.sig = "d";

      var nowIsMonotonic = 1;

      var checkWasiClock = (clock_id) => clock_id >= 0 && clock_id <= 3;

      var INT53_MAX = 9007199254740992;

      var INT53_MIN = -9007199254740992;

      var bigintToI53Checked = (num) =>
        (num < INT53_MIN || num > INT53_MAX) ? NaN : Number(num);

      function _clock_time_get(clk_id, ignored_precision, ptime) {
        ignored_precision = bigintToI53Checked(ignored_precision);
        if (!checkWasiClock(clk_id)) {
          return 28;
        }
        var now;
        // all wasi clocks but realtime are monotonic
        if (clk_id === 0) {
          now = _emscripten_date_now();
        } else if (nowIsMonotonic) {
          now = _emscripten_get_now();
        } else {
          return 52;
        }
        // "now" is in ms, and wasi times are in ns.
        var nsec = Math.round(now * 1e3 * 1e3);
        HEAP64[ptime >> 3] = BigInt(nsec);
        return 0;
      }

      _clock_time_get.sig = "iijp";

      var getHeapMax = () =>
        // Stay one Wasm page short of 4GB: while e.g. Chrome is able to allocate
        // full 4GB Wasm memories, the size will wrap back to 0 bytes in Wasm side
        // for any code that deals with heap sizes, which would require special
        // casing all heap size related code to treat 0 specially.
        2147483648;

      var growMemory = (size) => {
        var b = wasmMemory.buffer;
        var pages = ((size - b.byteLength + 65535) / 65536) | 0;
        try {
          // round size grow request up to wasm page size (fixed 64KB per spec)
          wasmMemory.grow(pages);
          // .grow() takes a delta compared to the previous size
          updateMemoryViews();
          return 1;
        } catch (e) {}
      };

      var _emscripten_resize_heap = (requestedSize) => {
        var oldSize = HEAPU8.length;
        // With CAN_ADDRESS_2GB or MEMORY64, pointers are already unsigned.
        requestedSize >>>= 0;
        // With multithreaded builds, races can happen (another thread might increase the size
        // in between), so return a failure, and let the caller retry.
        // Memory resize rules:
        // 1.  Always increase heap size to at least the requested size, rounded up
        //     to next page multiple.
        // 2a. If MEMORY_GROWTH_LINEAR_STEP == -1, excessively resize the heap
        //     geometrically: increase the heap size according to
        //     MEMORY_GROWTH_GEOMETRIC_STEP factor (default +20%), At most
        //     overreserve by MEMORY_GROWTH_GEOMETRIC_CAP bytes (default 96MB).
        // 2b. If MEMORY_GROWTH_LINEAR_STEP != -1, excessively resize the heap
        //     linearly: increase the heap size by at least
        //     MEMORY_GROWTH_LINEAR_STEP bytes.
        // 3.  Max size for the heap is capped at 2048MB-WASM_PAGE_SIZE, or by
        //     MAXIMUM_MEMORY, or by ASAN limit, depending on which is smallest
        // 4.  If we were unable to allocate as much memory, it may be due to
        //     over-eager decision to excessively reserve due to (3) above.
        //     Hence if an allocation fails, cut down on the amount of excess
        //     growth, in an attempt to succeed to perform a smaller allocation.
        // A limit is set for how much we can grow. We should not exceed that
        // (the wasm binary specifies it, so if we tried, we'd fail anyhow).
        var maxHeapSize = getHeapMax();
        if (requestedSize > maxHeapSize) {
          return false;
        }
        // Loop through potential heap size increases. If we attempt a too eager
        // reservation that fails, cut down on the attempted size and reserve a
        // smaller bump instead. (max 3 times, chosen somewhat arbitrarily)
        for (var cutDown = 1; cutDown <= 4; cutDown *= 2) {
          var overGrownHeapSize = oldSize * (1 + .2 / cutDown);
          // ensure geometric growth
          // but limit overreserving (default to capping at +96MB overgrowth at most)
          overGrownHeapSize = Math.min(
            overGrownHeapSize,
            requestedSize + 100663296,
          );
          var newSize = Math.min(
            maxHeapSize,
            alignMemory(Math.max(requestedSize, overGrownHeapSize), 65536),
          );
          var replacement = growMemory(newSize);
          if (replacement) {
            return true;
          }
        }
        return false;
      };

      _emscripten_resize_heap.sig = "ip";

      var _fd_close = (fd) => 52;

      _fd_close.sig = "ii";

      function _fd_seek(fd, offset, whence, newOffset) {
        offset = bigintToI53Checked(offset);
        return 70;
      }

      _fd_seek.sig = "iijip";

      var printCharBuffers = [null, [], []];

      var printChar = (stream, curr) => {
        var buffer = printCharBuffers[stream];
        if (curr === 0 || curr === 10) {
          (stream === 1 ? out : err)(UTF8ArrayToString(buffer));
          buffer.length = 0;
        } else {
          buffer.push(curr);
        }
      };

      var flush_NO_FILESYSTEM = () => {
        // flush anything remaining in the buffers during shutdown
        if (printCharBuffers[1].length) printChar(1, 10);
        if (printCharBuffers[2].length) printChar(2, 10);
      };

      var SYSCALLS = {
        varargs: undefined,
        getStr(ptr) {
          var ret = UTF8ToString(ptr);
          return ret;
        },
      };

      var _fd_write = (fd, iov, iovcnt, pnum) => {
        // hack to support printf in SYSCALLS_REQUIRE_FILESYSTEM=0
        var num = 0;
        for (var i = 0; i < iovcnt; i++) {
          var ptr = LE_HEAP_LOAD_U32((iov >> 2) * 4);
          var len = LE_HEAP_LOAD_U32(((iov + (4)) >> 2) * 4);
          iov += 8;
          for (var j = 0; j < len; j++) {
            printChar(fd, HEAPU8[ptr + j]);
          }
          num += len;
        }
        LE_HEAP_STORE_U32((pnum >> 2) * 4, num);
        return 0;
      };

      _fd_write.sig = "iippp";

      function _tree_sitter_log_callback(isLexMessage, messageAddress) {
        if (Module.currentLogCallback) {
          const message = UTF8ToString(messageAddress);
          Module.currentLogCallback(message, isLexMessage !== 0);
        }
      }

      function _tree_sitter_parse_callback(
        inputBufferAddress,
        index,
        row,
        column,
        lengthAddress,
      ) {
        const INPUT_BUFFER_SIZE = 10 * 1024;
        const string = Module.currentParseCallback(index, {
          row,
          column,
        });
        if (typeof string === "string") {
          setValue(lengthAddress, string.length, "i32");
          stringToUTF16(string, inputBufferAddress, INPUT_BUFFER_SIZE);
        } else {
          setValue(lengthAddress, 0, "i32");
        }
      }

      function _tree_sitter_progress_callback(currentOffset, hasError) {
        if (Module.currentProgressCallback) {
          return Module.currentProgressCallback({
            currentOffset,
            hasError,
          });
        }
        return false;
      }

      function _tree_sitter_query_progress_callback(currentOffset) {
        if (Module.currentQueryProgressCallback) {
          return Module.currentQueryProgressCallback({
            currentOffset,
          });
        }
        return false;
      }

      var runtimeKeepaliveCounter = 0;

      var keepRuntimeAlive = () => noExitRuntime || runtimeKeepaliveCounter > 0;

      var _proc_exit = (code) => {
        EXITSTATUS = code;
        if (!keepRuntimeAlive()) {
          Module["onExit"]?.(code);
          ABORT = true;
        }
        quit_(code, new ExitStatus(code));
      };

      _proc_exit.sig = "vi";

      /** @param {boolean|number=} implicit */ var exitJS = (
        status,
        implicit,
      ) => {
        EXITSTATUS = status;
        _proc_exit(status);
      };

      var handleException = (e) => {
        // Certain exception types we do not treat as errors since they are used for
        // internal control flow.
        // 1. ExitStatus, which is thrown by exit()
        // 2. "unwind", which is thrown by emscripten_unwind_to_js_event_loop() and others
        //    that wish to return to JS event loop.
        if (e instanceof ExitStatus || e == "unwind") {
          return EXITSTATUS;
        }
        quit_(1, e);
      };

      var lengthBytesUTF8 = (str) => {
        var len = 0;
        for (var i = 0; i < str.length; ++i) {
          // Gotcha: charCodeAt returns a 16-bit word that is a UTF-16 encoded code
          // unit, not a Unicode code point of the character! So decode
          // UTF16->UTF32->UTF8.
          // See http://unicode.org/faq/utf_bom.html#utf16-3
          var c = str.charCodeAt(i);
          // possibly a lead surrogate
          if (c <= 127) {
            len++;
          } else if (c <= 2047) {
            len += 2;
          } else if (c >= 55296 && c <= 57343) {
            len += 4;
            ++i;
          } else {
            len += 3;
          }
        }
        return len;
      };

      var stringToUTF8Array = (str, heap, outIdx, maxBytesToWrite) => {
        // Parameter maxBytesToWrite is not optional. Negative values, 0, null,
        // undefined and false each don't write out any bytes.
        if (!(maxBytesToWrite > 0)) return 0;
        var startIdx = outIdx;
        var endIdx = outIdx + maxBytesToWrite - 1;
        // -1 for string null terminator.
        for (var i = 0; i < str.length; ++i) {
          // Gotcha: charCodeAt returns a 16-bit word that is a UTF-16 encoded code
          // unit, not a Unicode code point of the character! So decode
          // UTF16->UTF32->UTF8.
          // See http://unicode.org/faq/utf_bom.html#utf16-3
          // For UTF8 byte structure, see http://en.wikipedia.org/wiki/UTF-8#Description
          // and https://www.ietf.org/rfc/rfc2279.txt
          // and https://tools.ietf.org/html/rfc3629
          var u = str.charCodeAt(i);
          // possibly a lead surrogate
          if (u >= 55296 && u <= 57343) {
            var u1 = str.charCodeAt(++i);
            u = 65536 + ((u & 1023) << 10) | (u1 & 1023);
          }
          if (u <= 127) {
            if (outIdx >= endIdx) break;
            heap[outIdx++] = u;
          } else if (u <= 2047) {
            if (outIdx + 1 >= endIdx) break;
            heap[outIdx++] = 192 | (u >> 6);
            heap[outIdx++] = 128 | (u & 63);
          } else if (u <= 65535) {
            if (outIdx + 2 >= endIdx) break;
            heap[outIdx++] = 224 | (u >> 12);
            heap[outIdx++] = 128 | ((u >> 6) & 63);
            heap[outIdx++] = 128 | (u & 63);
          } else {
            if (outIdx + 3 >= endIdx) break;
            heap[outIdx++] = 240 | (u >> 18);
            heap[outIdx++] = 128 | ((u >> 12) & 63);
            heap[outIdx++] = 128 | ((u >> 6) & 63);
            heap[outIdx++] = 128 | (u & 63);
          }
        }
        // Null-terminate the pointer to the buffer.
        heap[outIdx] = 0;
        return outIdx - startIdx;
      };

      var stringToUTF8 = (str, outPtr, maxBytesToWrite) =>
        stringToUTF8Array(str, HEAPU8, outPtr, maxBytesToWrite);

      var stackAlloc = (sz) => __emscripten_stack_alloc(sz);

      var stringToUTF8OnStack = (str) => {
        var size = lengthBytesUTF8(str) + 1;
        var ret = stackAlloc(size);
        stringToUTF8(str, ret, size);
        return ret;
      };

      var AsciiToString = (ptr) => {
        var str = "";
        while (1) {
          var ch = HEAPU8[ptr++];
          if (!ch) return str;
          str += String.fromCharCode(ch);
        }
      };

      var stringToUTF16 = (str, outPtr, maxBytesToWrite) => {
        // Backwards compatibility: if max bytes is not specified, assume unsafe unbounded write is allowed.
        maxBytesToWrite ??= 2147483647;
        if (maxBytesToWrite < 2) return 0;
        maxBytesToWrite -= 2;
        // Null terminator.
        var startPtr = outPtr;
        var numCharsToWrite = (maxBytesToWrite < str.length * 2)
          ? (maxBytesToWrite / 2)
          : str.length;
        for (var i = 0; i < numCharsToWrite; ++i) {
          // charCodeAt returns a UTF-16 encoded code unit, so it can be directly written to the HEAP.
          var codeUnit = str.charCodeAt(i);
          // possibly a lead surrogate
          LE_HEAP_STORE_I16((outPtr >> 1) * 2, codeUnit);
          outPtr += 2;
        }
        // Null-terminate the pointer to the HEAP.
        LE_HEAP_STORE_I16((outPtr >> 1) * 2, 0);
        return outPtr - startPtr;
      };

      LE_ATOMICS_NATIVE_BYTE_ORDER =
        (new Int8Array(new Int16Array([1]).buffer)[0] === 1)
          ? [/* little endian */ (x) => x, (x) => x, undefined, (x) => x]
          : [
            /* big endian */ (x) => x,
            (x) => (((x & 65280) << 8) | ((x & 255) << 24)) >> 16,
            undefined,
            (x) =>
              ((x >> 24) & 255) | ((x >> 8) & 65280) | ((x & 65280) << 8) |
              ((x & 255) << 24),
          ];

      function LE_HEAP_UPDATE() {
        HEAPU16.unsigned = (x) => x & 65535;
        HEAPU32.unsigned = (x) => x >>> 0;
      }

      // End JS library code
      var ASM_CONSTS = {};

      var wasmImports = {
        /** @export */ __heap_base: ___heap_base,
        /** @export */ __indirect_function_table: wasmTable,
        /** @export */ __memory_base: ___memory_base,
        /** @export */ __stack_pointer: ___stack_pointer,
        /** @export */ __table_base: ___table_base,
        /** @export */ _abort_js: __abort_js,
        /** @export */ clock_time_get: _clock_time_get,
        /** @export */ emscripten_resize_heap: _emscripten_resize_heap,
        /** @export */ fd_close: _fd_close,
        /** @export */ fd_seek: _fd_seek,
        /** @export */ fd_write: _fd_write,
        /** @export */ memory: wasmMemory,
        /** @export */ tree_sitter_log_callback: _tree_sitter_log_callback,
        /** @export */ tree_sitter_parse_callback: _tree_sitter_parse_callback,
        /** @export */ tree_sitter_progress_callback:
          _tree_sitter_progress_callback,
        /** @export */ tree_sitter_query_progress_callback:
          _tree_sitter_query_progress_callback,
      };

      var wasmExports = await createWasm();

      var ___wasm_call_ctors = wasmExports["__wasm_call_ctors"];

      var _malloc = Module["_malloc"] = wasmExports["malloc"];

      var _calloc = Module["_calloc"] = wasmExports["calloc"];

      var _realloc = Module["_realloc"] = wasmExports["realloc"];

      var _free = Module["_free"] = wasmExports["free"];

      var _memcmp = Module["_memcmp"] = wasmExports["memcmp"];

      var _ts_language_symbol_count = Module["_ts_language_symbol_count"] =
        wasmExports["ts_language_symbol_count"];

      var _ts_language_state_count = Module["_ts_language_state_count"] =
        wasmExports["ts_language_state_count"];

      var _ts_language_version = Module["_ts_language_version"] =
        wasmExports["ts_language_version"];

      var _ts_language_abi_version = Module["_ts_language_abi_version"] =
        wasmExports["ts_language_abi_version"];

      var _ts_language_metadata = Module["_ts_language_metadata"] =
        wasmExports["ts_language_metadata"];

      var _ts_language_name = Module["_ts_language_name"] =
        wasmExports["ts_language_name"];

      var _ts_language_field_count = Module["_ts_language_field_count"] =
        wasmExports["ts_language_field_count"];

      var _ts_language_next_state = Module["_ts_language_next_state"] =
        wasmExports["ts_language_next_state"];

      var _ts_language_symbol_name = Module["_ts_language_symbol_name"] =
        wasmExports["ts_language_symbol_name"];

      var _ts_language_symbol_for_name =
        Module["_ts_language_symbol_for_name"] =
          wasmExports["ts_language_symbol_for_name"];

      var _strncmp = Module["_strncmp"] = wasmExports["strncmp"];

      var _ts_language_symbol_type = Module["_ts_language_symbol_type"] =
        wasmExports["ts_language_symbol_type"];

      var _ts_language_field_name_for_id =
        Module["_ts_language_field_name_for_id"] =
          wasmExports["ts_language_field_name_for_id"];

      var _ts_lookahead_iterator_new = Module["_ts_lookahead_iterator_new"] =
        wasmExports["ts_lookahead_iterator_new"];

      var _ts_lookahead_iterator_delete =
        Module["_ts_lookahead_iterator_delete"] =
          wasmExports["ts_lookahead_iterator_delete"];

      var _ts_lookahead_iterator_reset_state =
        Module["_ts_lookahead_iterator_reset_state"] =
          wasmExports["ts_lookahead_iterator_reset_state"];

      var _ts_lookahead_iterator_reset =
        Module["_ts_lookahead_iterator_reset"] =
          wasmExports["ts_lookahead_iterator_reset"];

      var _ts_lookahead_iterator_next = Module["_ts_lookahead_iterator_next"] =
        wasmExports["ts_lookahead_iterator_next"];

      var _ts_lookahead_iterator_current_symbol =
        Module["_ts_lookahead_iterator_current_symbol"] =
          wasmExports["ts_lookahead_iterator_current_symbol"];

      var _ts_parser_delete = Module["_ts_parser_delete"] =
        wasmExports["ts_parser_delete"];

      var _ts_parser_reset = Module["_ts_parser_reset"] =
        wasmExports["ts_parser_reset"];

      var _ts_parser_set_language = Module["_ts_parser_set_language"] =
        wasmExports["ts_parser_set_language"];

      var _ts_parser_timeout_micros = Module["_ts_parser_timeout_micros"] =
        wasmExports["ts_parser_timeout_micros"];

      var _ts_parser_set_timeout_micros =
        Module["_ts_parser_set_timeout_micros"] =
          wasmExports["ts_parser_set_timeout_micros"];

      var _ts_parser_set_included_ranges =
        Module["_ts_parser_set_included_ranges"] =
          wasmExports["ts_parser_set_included_ranges"];

      var _ts_query_new = Module["_ts_query_new"] = wasmExports["ts_query_new"];

      var _ts_query_delete = Module["_ts_query_delete"] =
        wasmExports["ts_query_delete"];

      var _iswspace = Module["_iswspace"] = wasmExports["iswspace"];

      var _iswalnum = Module["_iswalnum"] = wasmExports["iswalnum"];

      var _ts_query_pattern_count = Module["_ts_query_pattern_count"] =
        wasmExports["ts_query_pattern_count"];

      var _ts_query_capture_count = Module["_ts_query_capture_count"] =
        wasmExports["ts_query_capture_count"];

      var _ts_query_string_count = Module["_ts_query_string_count"] =
        wasmExports["ts_query_string_count"];

      var _ts_query_capture_name_for_id =
        Module["_ts_query_capture_name_for_id"] =
          wasmExports["ts_query_capture_name_for_id"];

      var _ts_query_capture_quantifier_for_id =
        Module["_ts_query_capture_quantifier_for_id"] =
          wasmExports["ts_query_capture_quantifier_for_id"];

      var _ts_query_string_value_for_id =
        Module["_ts_query_string_value_for_id"] =
          wasmExports["ts_query_string_value_for_id"];

      var _ts_query_predicates_for_pattern =
        Module["_ts_query_predicates_for_pattern"] =
          wasmExports["ts_query_predicates_for_pattern"];

      var _ts_query_start_byte_for_pattern =
        Module["_ts_query_start_byte_for_pattern"] =
          wasmExports["ts_query_start_byte_for_pattern"];

      var _ts_query_end_byte_for_pattern =
        Module["_ts_query_end_byte_for_pattern"] =
          wasmExports["ts_query_end_byte_for_pattern"];

      var _ts_query_is_pattern_rooted = Module["_ts_query_is_pattern_rooted"] =
        wasmExports["ts_query_is_pattern_rooted"];

      var _ts_query_is_pattern_non_local =
        Module["_ts_query_is_pattern_non_local"] =
          wasmExports["ts_query_is_pattern_non_local"];

      var _ts_query_is_pattern_guaranteed_at_step =
        Module["_ts_query_is_pattern_guaranteed_at_step"] =
          wasmExports["ts_query_is_pattern_guaranteed_at_step"];

      var _ts_query_disable_capture = Module["_ts_query_disable_capture"] =
        wasmExports["ts_query_disable_capture"];

      var _ts_query_disable_pattern = Module["_ts_query_disable_pattern"] =
        wasmExports["ts_query_disable_pattern"];

      var _ts_tree_copy = Module["_ts_tree_copy"] = wasmExports["ts_tree_copy"];

      var _ts_tree_delete = Module["_ts_tree_delete"] =
        wasmExports["ts_tree_delete"];

      var _ts_init = Module["_ts_init"] = wasmExports["ts_init"];

      var _ts_parser_new_wasm = Module["_ts_parser_new_wasm"] =
        wasmExports["ts_parser_new_wasm"];

      var _ts_parser_enable_logger_wasm =
        Module["_ts_parser_enable_logger_wasm"] =
          wasmExports["ts_parser_enable_logger_wasm"];

      var _ts_parser_parse_wasm = Module["_ts_parser_parse_wasm"] =
        wasmExports["ts_parser_parse_wasm"];

      var _ts_parser_included_ranges_wasm =
        Module["_ts_parser_included_ranges_wasm"] =
          wasmExports["ts_parser_included_ranges_wasm"];

      var _ts_language_type_is_named_wasm =
        Module["_ts_language_type_is_named_wasm"] =
          wasmExports["ts_language_type_is_named_wasm"];

      var _ts_language_type_is_visible_wasm =
        Module["_ts_language_type_is_visible_wasm"] =
          wasmExports["ts_language_type_is_visible_wasm"];

      var _ts_language_supertypes_wasm =
        Module["_ts_language_supertypes_wasm"] =
          wasmExports["ts_language_supertypes_wasm"];

      var _ts_language_subtypes_wasm = Module["_ts_language_subtypes_wasm"] =
        wasmExports["ts_language_subtypes_wasm"];

      var _ts_tree_root_node_wasm = Module["_ts_tree_root_node_wasm"] =
        wasmExports["ts_tree_root_node_wasm"];

      var _ts_tree_root_node_with_offset_wasm =
        Module["_ts_tree_root_node_with_offset_wasm"] =
          wasmExports["ts_tree_root_node_with_offset_wasm"];

      var _ts_tree_edit_wasm = Module["_ts_tree_edit_wasm"] =
        wasmExports["ts_tree_edit_wasm"];

      var _ts_tree_included_ranges_wasm =
        Module["_ts_tree_included_ranges_wasm"] =
          wasmExports["ts_tree_included_ranges_wasm"];

      var _ts_tree_get_changed_ranges_wasm =
        Module["_ts_tree_get_changed_ranges_wasm"] =
          wasmExports["ts_tree_get_changed_ranges_wasm"];

      var _ts_tree_cursor_new_wasm = Module["_ts_tree_cursor_new_wasm"] =
        wasmExports["ts_tree_cursor_new_wasm"];

      var _ts_tree_cursor_copy_wasm = Module["_ts_tree_cursor_copy_wasm"] =
        wasmExports["ts_tree_cursor_copy_wasm"];

      var _ts_tree_cursor_delete_wasm = Module["_ts_tree_cursor_delete_wasm"] =
        wasmExports["ts_tree_cursor_delete_wasm"];

      var _ts_tree_cursor_reset_wasm = Module["_ts_tree_cursor_reset_wasm"] =
        wasmExports["ts_tree_cursor_reset_wasm"];

      var _ts_tree_cursor_reset_to_wasm =
        Module["_ts_tree_cursor_reset_to_wasm"] =
          wasmExports["ts_tree_cursor_reset_to_wasm"];

      var _ts_tree_cursor_goto_first_child_wasm =
        Module["_ts_tree_cursor_goto_first_child_wasm"] =
          wasmExports["ts_tree_cursor_goto_first_child_wasm"];

      var _ts_tree_cursor_goto_last_child_wasm =
        Module["_ts_tree_cursor_goto_last_child_wasm"] =
          wasmExports["ts_tree_cursor_goto_last_child_wasm"];

      var _ts_tree_cursor_goto_first_child_for_index_wasm =
        Module["_ts_tree_cursor_goto_first_child_for_index_wasm"] =
          wasmExports["ts_tree_cursor_goto_first_child_for_index_wasm"];

      var _ts_tree_cursor_goto_first_child_for_position_wasm =
        Module["_ts_tree_cursor_goto_first_child_for_position_wasm"] =
          wasmExports["ts_tree_cursor_goto_first_child_for_position_wasm"];

      var _ts_tree_cursor_goto_next_sibling_wasm =
        Module["_ts_tree_cursor_goto_next_sibling_wasm"] =
          wasmExports["ts_tree_cursor_goto_next_sibling_wasm"];

      var _ts_tree_cursor_goto_previous_sibling_wasm =
        Module["_ts_tree_cursor_goto_previous_sibling_wasm"] =
          wasmExports["ts_tree_cursor_goto_previous_sibling_wasm"];

      var _ts_tree_cursor_goto_descendant_wasm =
        Module["_ts_tree_cursor_goto_descendant_wasm"] =
          wasmExports["ts_tree_cursor_goto_descendant_wasm"];

      var _ts_tree_cursor_goto_parent_wasm =
        Module["_ts_tree_cursor_goto_parent_wasm"] =
          wasmExports["ts_tree_cursor_goto_parent_wasm"];

      var _ts_tree_cursor_current_node_type_id_wasm =
        Module["_ts_tree_cursor_current_node_type_id_wasm"] =
          wasmExports["ts_tree_cursor_current_node_type_id_wasm"];

      var _ts_tree_cursor_current_node_state_id_wasm =
        Module["_ts_tree_cursor_current_node_state_id_wasm"] =
          wasmExports["ts_tree_cursor_current_node_state_id_wasm"];

      var _ts_tree_cursor_current_node_is_named_wasm =
        Module["_ts_tree_cursor_current_node_is_named_wasm"] =
          wasmExports["ts_tree_cursor_current_node_is_named_wasm"];

      var _ts_tree_cursor_current_node_is_missing_wasm =
        Module["_ts_tree_cursor_current_node_is_missing_wasm"] =
          wasmExports["ts_tree_cursor_current_node_is_missing_wasm"];

      var _ts_tree_cursor_current_node_id_wasm =
        Module["_ts_tree_cursor_current_node_id_wasm"] =
          wasmExports["ts_tree_cursor_current_node_id_wasm"];

      var _ts_tree_cursor_start_position_wasm =
        Module["_ts_tree_cursor_start_position_wasm"] =
          wasmExports["ts_tree_cursor_start_position_wasm"];

      var _ts_tree_cursor_end_position_wasm =
        Module["_ts_tree_cursor_end_position_wasm"] =
          wasmExports["ts_tree_cursor_end_position_wasm"];

      var _ts_tree_cursor_start_index_wasm =
        Module["_ts_tree_cursor_start_index_wasm"] =
          wasmExports["ts_tree_cursor_start_index_wasm"];

      var _ts_tree_cursor_end_index_wasm =
        Module["_ts_tree_cursor_end_index_wasm"] =
          wasmExports["ts_tree_cursor_end_index_wasm"];

      var _ts_tree_cursor_current_field_id_wasm =
        Module["_ts_tree_cursor_current_field_id_wasm"] =
          wasmExports["ts_tree_cursor_current_field_id_wasm"];

      var _ts_tree_cursor_current_depth_wasm =
        Module["_ts_tree_cursor_current_depth_wasm"] =
          wasmExports["ts_tree_cursor_current_depth_wasm"];

      var _ts_tree_cursor_current_descendant_index_wasm =
        Module["_ts_tree_cursor_current_descendant_index_wasm"] =
          wasmExports["ts_tree_cursor_current_descendant_index_wasm"];

      var _ts_tree_cursor_current_node_wasm =
        Module["_ts_tree_cursor_current_node_wasm"] =
          wasmExports["ts_tree_cursor_current_node_wasm"];

      var _ts_node_symbol_wasm = Module["_ts_node_symbol_wasm"] =
        wasmExports["ts_node_symbol_wasm"];

      var _ts_node_field_name_for_child_wasm =
        Module["_ts_node_field_name_for_child_wasm"] =
          wasmExports["ts_node_field_name_for_child_wasm"];

      var _ts_node_field_name_for_named_child_wasm =
        Module["_ts_node_field_name_for_named_child_wasm"] =
          wasmExports["ts_node_field_name_for_named_child_wasm"];

      var _ts_node_children_by_field_id_wasm =
        Module["_ts_node_children_by_field_id_wasm"] =
          wasmExports["ts_node_children_by_field_id_wasm"];

      var _ts_node_first_child_for_byte_wasm =
        Module["_ts_node_first_child_for_byte_wasm"] =
          wasmExports["ts_node_first_child_for_byte_wasm"];

      var _ts_node_first_named_child_for_byte_wasm =
        Module["_ts_node_first_named_child_for_byte_wasm"] =
          wasmExports["ts_node_first_named_child_for_byte_wasm"];

      var _ts_node_grammar_symbol_wasm =
        Module["_ts_node_grammar_symbol_wasm"] =
          wasmExports["ts_node_grammar_symbol_wasm"];

      var _ts_node_child_count_wasm = Module["_ts_node_child_count_wasm"] =
        wasmExports["ts_node_child_count_wasm"];

      var _ts_node_named_child_count_wasm =
        Module["_ts_node_named_child_count_wasm"] =
          wasmExports["ts_node_named_child_count_wasm"];

      var _ts_node_child_wasm = Module["_ts_node_child_wasm"] =
        wasmExports["ts_node_child_wasm"];

      var _ts_node_named_child_wasm = Module["_ts_node_named_child_wasm"] =
        wasmExports["ts_node_named_child_wasm"];

      var _ts_node_child_by_field_id_wasm =
        Module["_ts_node_child_by_field_id_wasm"] =
          wasmExports["ts_node_child_by_field_id_wasm"];

      var _ts_node_next_sibling_wasm = Module["_ts_node_next_sibling_wasm"] =
        wasmExports["ts_node_next_sibling_wasm"];

      var _ts_node_prev_sibling_wasm = Module["_ts_node_prev_sibling_wasm"] =
        wasmExports["ts_node_prev_sibling_wasm"];

      var _ts_node_next_named_sibling_wasm =
        Module["_ts_node_next_named_sibling_wasm"] =
          wasmExports["ts_node_next_named_sibling_wasm"];

      var _ts_node_prev_named_sibling_wasm =
        Module["_ts_node_prev_named_sibling_wasm"] =
          wasmExports["ts_node_prev_named_sibling_wasm"];

      var _ts_node_descendant_count_wasm =
        Module["_ts_node_descendant_count_wasm"] =
          wasmExports["ts_node_descendant_count_wasm"];

      var _ts_node_parent_wasm = Module["_ts_node_parent_wasm"] =
        wasmExports["ts_node_parent_wasm"];

      var _ts_node_child_with_descendant_wasm =
        Module["_ts_node_child_with_descendant_wasm"] =
          wasmExports["ts_node_child_with_descendant_wasm"];

      var _ts_node_descendant_for_index_wasm =
        Module["_ts_node_descendant_for_index_wasm"] =
          wasmExports["ts_node_descendant_for_index_wasm"];

      var _ts_node_named_descendant_for_index_wasm =
        Module["_ts_node_named_descendant_for_index_wasm"] =
          wasmExports["ts_node_named_descendant_for_index_wasm"];

      var _ts_node_descendant_for_position_wasm =
        Module["_ts_node_descendant_for_position_wasm"] =
          wasmExports["ts_node_descendant_for_position_wasm"];

      var _ts_node_named_descendant_for_position_wasm =
        Module["_ts_node_named_descendant_for_position_wasm"] =
          wasmExports["ts_node_named_descendant_for_position_wasm"];

      var _ts_node_start_point_wasm = Module["_ts_node_start_point_wasm"] =
        wasmExports["ts_node_start_point_wasm"];

      var _ts_node_end_point_wasm = Module["_ts_node_end_point_wasm"] =
        wasmExports["ts_node_end_point_wasm"];

      var _ts_node_start_index_wasm = Module["_ts_node_start_index_wasm"] =
        wasmExports["ts_node_start_index_wasm"];

      var _ts_node_end_index_wasm = Module["_ts_node_end_index_wasm"] =
        wasmExports["ts_node_end_index_wasm"];

      var _ts_node_to_string_wasm = Module["_ts_node_to_string_wasm"] =
        wasmExports["ts_node_to_string_wasm"];

      var _ts_node_children_wasm = Module["_ts_node_children_wasm"] =
        wasmExports["ts_node_children_wasm"];

      var _ts_node_named_children_wasm =
        Module["_ts_node_named_children_wasm"] =
          wasmExports["ts_node_named_children_wasm"];

      var _ts_node_descendants_of_type_wasm =
        Module["_ts_node_descendants_of_type_wasm"] =
          wasmExports["ts_node_descendants_of_type_wasm"];

      var _ts_node_is_named_wasm = Module["_ts_node_is_named_wasm"] =
        wasmExports["ts_node_is_named_wasm"];

      var _ts_node_has_changes_wasm = Module["_ts_node_has_changes_wasm"] =
        wasmExports["ts_node_has_changes_wasm"];

      var _ts_node_has_error_wasm = Module["_ts_node_has_error_wasm"] =
        wasmExports["ts_node_has_error_wasm"];

      var _ts_node_is_error_wasm = Module["_ts_node_is_error_wasm"] =
        wasmExports["ts_node_is_error_wasm"];

      var _ts_node_is_missing_wasm = Module["_ts_node_is_missing_wasm"] =
        wasmExports["ts_node_is_missing_wasm"];

      var _ts_node_is_extra_wasm = Module["_ts_node_is_extra_wasm"] =
        wasmExports["ts_node_is_extra_wasm"];

      var _ts_node_parse_state_wasm = Module["_ts_node_parse_state_wasm"] =
        wasmExports["ts_node_parse_state_wasm"];

      var _ts_node_next_parse_state_wasm =
        Module["_ts_node_next_parse_state_wasm"] =
          wasmExports["ts_node_next_parse_state_wasm"];

      var _ts_query_matches_wasm = Module["_ts_query_matches_wasm"] =
        wasmExports["ts_query_matches_wasm"];

      var _ts_query_captures_wasm = Module["_ts_query_captures_wasm"] =
        wasmExports["ts_query_captures_wasm"];

      var _memset = Module["_memset"] = wasmExports["memset"];

      var _memcpy = Module["_memcpy"] = wasmExports["memcpy"];

      var _memmove = Module["_memmove"] = wasmExports["memmove"];

      var _iswalpha = Module["_iswalpha"] = wasmExports["iswalpha"];

      var _iswblank = Module["_iswblank"] = wasmExports["iswblank"];

      var _iswdigit = Module["_iswdigit"] = wasmExports["iswdigit"];

      var _iswlower = Module["_iswlower"] = wasmExports["iswlower"];

      var _iswupper = Module["_iswupper"] = wasmExports["iswupper"];

      var _iswxdigit = Module["_iswxdigit"] = wasmExports["iswxdigit"];

      var _memchr = Module["_memchr"] = wasmExports["memchr"];

      var _strlen = Module["_strlen"] = wasmExports["strlen"];

      var _strcmp = Module["_strcmp"] = wasmExports["strcmp"];

      var _strncat = Module["_strncat"] = wasmExports["strncat"];

      var _strncpy = Module["_strncpy"] = wasmExports["strncpy"];

      var _towlower = Module["_towlower"] = wasmExports["towlower"];

      var _towupper = Module["_towupper"] = wasmExports["towupper"];

      var _setThrew = wasmExports["setThrew"];

      var __emscripten_stack_restore = wasmExports["_emscripten_stack_restore"];

      var __emscripten_stack_alloc = wasmExports["_emscripten_stack_alloc"];

      var _emscripten_stack_get_current =
        wasmExports["emscripten_stack_get_current"];

      var ___wasm_apply_data_relocs = wasmExports["__wasm_apply_data_relocs"];

      // include: postamble.js
      // === Auto-generated postamble setup entry stuff ===
      Module["setValue"] = setValue;

      Module["getValue"] = getValue;

      Module["UTF8ToString"] = UTF8ToString;

      Module["stringToUTF8"] = stringToUTF8;

      Module["lengthBytesUTF8"] = lengthBytesUTF8;

      Module["AsciiToString"] = AsciiToString;

      Module["stringToUTF16"] = stringToUTF16;

      Module["loadWebAssemblyModule"] = loadWebAssemblyModule;

      function callMain(args = []) {
        var entryFunction = resolveGlobalSymbol("main").sym;
        // Main modules can't tell if they have main() at compile time, since it may
        // arrive from a dynamic library.
        if (!entryFunction) return;
        args.unshift(thisProgram);
        var argc = args.length;
        var argv = stackAlloc((argc + 1) * 4);
        var argv_ptr = argv;
        args.forEach((arg) => {
          LE_HEAP_STORE_U32((argv_ptr >> 2) * 4, stringToUTF8OnStack(arg));
          argv_ptr += 4;
        });
        LE_HEAP_STORE_U32((argv_ptr >> 2) * 4, 0);
        try {
          var ret = entryFunction(argc, argv);
          // if we're not running an evented main loop, it's time to exit
          exitJS(ret, /* implicit = */ true);
          return ret;
        } catch (e) {
          return handleException(e);
        }
      }

      function run(args = arguments_) {
        if (runDependencies > 0) {
          dependenciesFulfilled = run;
          return;
        }
        preRun();
        // a preRun added a dependency, run will be called later
        if (runDependencies > 0) {
          dependenciesFulfilled = run;
          return;
        }
        function doRun() {
          // run may have just been called through dependencies being fulfilled just in this very frame,
          // or while the async setStatus time below was happening
          Module["calledRun"] = true;
          if (ABORT) return;
          initRuntime();
          preMain();
          readyPromiseResolve(Module);
          Module["onRuntimeInitialized"]?.();
          var noInitialRun = Module["noInitialRun"];
          if (!noInitialRun) callMain(args);
          postRun();
        }
        if (Module["setStatus"]) {
          Module["setStatus"]("Running...");
          setTimeout(() => {
            setTimeout(() => Module["setStatus"](""), 1);
            doRun();
          }, 1);
        } else {
          doRun();
        }
      }

      if (Module["preInit"]) {
        if (typeof Module["preInit"] == "function") {
          Module["preInit"] = [Module["preInit"]];
        }
        while (Module["preInit"].length > 0) {
          Module["preInit"].pop()();
        }
      }

      run();

      // end include: postamble.js
      // include: postamble_modularize.js
      // In MODULARIZE mode we wrap the generated code in a factory function
      // and return either the Module itself, or a promise of the module.
      // We assign to the `moduleRtn` global here and configure closure to see
      // this as and extern so it won't get minified.
      moduleRtn = readyPromise;

      return moduleRtn;
    }
  );
})();
export default Module;
