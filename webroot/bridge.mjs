// KernelSU's callback contract, verified against the official kernelsu npm SDK 3.3.0.
let sequence = 0;

export function encodeRequest(request) {
  const json = JSON.stringify(request);
  const bytes = new TextEncoder().encode(json);
  if (bytes.length > 64 * 1024)
    throw new Error("Request exceeds the 64 KiB settings limit.");
  let binary = "";
  for (const byte of bytes) binary += String.fromCharCode(byte);
  return btoa(binary);
}

export function createBridge(scope = globalThis, { timeout = 30000 } = {}) {
  const supported = typeof scope.ksu?.exec === "function";
  return {
    supported,
    async request(request) {
      if (!supported)
        throw new Error(
          "Unsupported environment. Open Sortify in a KernelSU manager with the WebUI callback bridge.",
        );
      const encoded = encodeRequest(request);
      // Only base64 enters shell text. The complete argument is quoted exactly once.
      const command = `sh /data/adb/modules/sortify/sortify.sh request '${encoded}'`;
      return new Promise((resolve, reject) => {
        const callback = `sortify_callback_${Date.now()}_${sequence++}`;
        const cleanup = () => {
          clearTimeout(timer);
          delete scope[callback];
        };
        const timer = setTimeout(() => {
          cleanup();
          reject(
            new Error(
              "Device request timed out. Its outcome is unknown; refresh status before retrying.",
            ),
          );
        }, timeout);
        scope[callback] = (errno, stdout, stderr) => {
          cleanup();
          try {
            const commandFailure = () =>
              new Error(
                `Device command exited with exit ${String(errno)}${stderr ? `: ${String(stderr).trim()}` : "."}`,
              );
            let response;
            try {
              response = JSON.parse(stdout);
            } catch {
              if (errno !== 0) throw commandFailure();
              throw new Error(
                "Device returned invalid JSON. Check the installed module version.",
              );
            }
            const validEnvelope =
              response &&
              typeof response === "object" &&
              typeof response.ok === "boolean";
            // The native CLI returns its structured failure on stdout and exits 1.
            if (validEnvelope && !response.ok)
              throw new Error(
                typeof response.error === "string" && response.error.trim()
                  ? response.error
                  : "Device request failed without an error message.",
              );
            if (errno !== 0) throw commandFailure();
            if (!validEnvelope)
              throw new Error("Device returned an invalid response envelope.");
            if (!Object.hasOwn(response, "data"))
              throw new Error(
                "Device returned an incomplete response envelope.",
              );
            resolve(response.data);
          } catch (error) {
            reject(error);
          }
        };
        try {
          scope.ksu.exec(command, "{}", callback);
        } catch (error) {
          cleanup();
          reject(
            new Error(`KernelSU bridge failed: ${error.message}`, {
              cause: error,
            }),
          );
        }
      });
    },
  };
}
