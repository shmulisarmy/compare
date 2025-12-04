const { execSync } = require("child_process");
const fs = require("fs");
const http = require("http");
const urlModule = require("url");

/**
 * Convert a value to a JSON string safe for terminal commands
 * @param {any} v
 * @returns {string}
 */
function toTerminalSafeJsonString(v) {
  return JSON.stringify(v);
}

/**
 * Compare expected and actual values using local `compare` binary if available,
 * otherwise fallback to HTTP request.
 * @param {any} expected
 * @param {any} actual
 * @param {string} [url]
 * @returns {Promise<{status: number, body: string, error?: Error}>}
 */
function compare(expected, actual, url = "") {
  return new Promise((resolve) => {
    let comparePath;
    try {
      comparePath = execSync("which compare", { stdio: ["pipe", "pipe", "ignore"] })
        .toString()
        .trim();
    } catch {
      comparePath = null;
    }

    if (comparePath && fs.existsSync(comparePath)) {
      const terminalCommand = `${comparePath} -expected='${toTerminalSafeJsonString(
        expected
      )}' -actual='${toTerminalSafeJsonString(actual)}'`;
      try {
        const output = execSync(terminalCommand, { encoding: "utf8" });
        resolve({ status: 0, body: output });
        return;
      } catch (err) {
        resolve({ status: 1, body: err.stdout + err.stderr, error: err });
        return;
      }
    } else {
      console.log("compare is not a command");
    }

    if (!url) url = "http://localhost:8080/compare";

    const parsedUrl = urlModule.parse(url);
    const payload = JSON.stringify({ expected, actual });

    const options = {
      hostname: parsedUrl.hostname,
      port: parsedUrl.port || 80,
      path: parsedUrl.path,
      method: "GET",
      headers: {
        "Content-Type": "application/json",
        "Content-Length": Buffer.byteLength(payload),
      },
    };

    const req = http.request(options, (res) => {
      let data = "";
      res.on("data", (chunk) => (data += chunk));
      res.on("end", () => resolve({ status: res.statusCode, body: data }));
    });

    req.on("error", (err) => resolve({ status: 0, body: "", error: err }));
    req.write(payload);
    req.end();
  });
}


compare({
    "name": "bob",
    "age": 30,
    "friends": [
        {
            "name": "charlie"
        },
        "bob",
        "gil",
    ]
}, {
    "name": "alice",
    "age": 30,
    "friends": [
        {
            "name": "charlie2"
        },
        "bob",
        "gil",
    ]
}).then(result => console.log(result.body));

module.exports = { compare, toTerminalSafeJsonString };
