// Copyright 2025 The Outline Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import { exec, execFile } from "node:child_process";
import { glob } from "glob";
import { promisify } from "node:util";
import chalk from "chalk";
import archiver from "archiver";
import fs from "node:fs";
import handlebars from "handlebars";
import path from "node:path";
import minimist from "minimist";

const OUTPUT_DIR = path.join(process.cwd(), "output");
const WRAPPER_APP_TEMPLATE_DIR = path.join(
  process.cwd(),
  "wrapper_app_project/template",
);

const DEFAULT_SMART_DIALER_CONFIG = JSON.stringify({
  dns: [{
    https: { name: "9.9.9.9" },
  }],
  tls: ["", "split:1", "split:2", "tlsfrag:1"],
});

export default async function main(
  {
    additionalDomains = [],
    appId,
    appName,
    entryUrl = "https://www.example.com",
    navigationUrl,
    output = OUTPUT_DIR,
    platform,
    smartDialerConfig = DEFAULT_SMART_DIALER_CONFIG,
  },
) {
  const WRAPPER_APP_OUTPUT_DIR = path.resolve(output, "wrapper_app_project");
  const WRAPPER_APP_OUTPUT_ZIP = path.resolve(
    output,
    "wrapper_app_project.zip",
  );

  const SDK_MOBILEPROXY_OUTPUT_DIR = path.resolve(output, "mobileproxy");
  const WRAPPER_APP_OUTPUT_SDK_MOBILEPROXY_DIR = path.resolve(
    WRAPPER_APP_OUTPUT_DIR,
    "mobileproxy",
  );

  if (!fs.existsSync(SDK_MOBILEPROXY_OUTPUT_DIR)) {
    console.log(
      `Building the Outline SDK mobileproxy library for ${platform}...`,
    );

    await promisify(execFile)("npm", [
      "run",
      "build:mobileproxy",
      platform,
      output,
    ], { shell: false });
  }

  const sourceFilepaths = await glob(
    path.join(WRAPPER_APP_TEMPLATE_DIR, "**", "*"),
    {
      nodir: true,
      dot: true,
    },
  );

  console.log("Building project from template...");

  let templateArguments;

  try {
    templateArguments = resolveTemplateArguments(platform, entryUrl, {
      additionalDomains,
      appId,
      appName,
      navigationUrl,
      smartDialerConfig,
    });
  } catch (cause) {
    throw new TypeError("Failed to resolve the project template arguments", {
      cause,
    });
  }

  for (const sourceFilepath of sourceFilepaths) {
    const destinationFilepath = path.join(
      WRAPPER_APP_OUTPUT_DIR,
      path.relative(
        WRAPPER_APP_TEMPLATE_DIR,
        sourceFilepath,
      ),
    );

    // ensure directory
    fs.mkdirSync(path.dirname(destinationFilepath), { recursive: true });

    if (!sourceFilepath.endsWith(".handlebars")) {
      fs.copyFileSync(sourceFilepath, destinationFilepath);
      continue;
    }

    const template = handlebars.compile(
      fs.readFileSync(sourceFilepath, "utf8"),
    );

    fs.writeFileSync(
      destinationFilepath.replace(/\.handlebars$/, ""),
      template(templateArguments),
      "utf8",
    );
  }

  console.log("Copying mobileproxy files into the project...");

  fs.cpSync(
    SDK_MOBILEPROXY_OUTPUT_DIR,
    WRAPPER_APP_OUTPUT_SDK_MOBILEPROXY_DIR,
    { recursive: true },
  );

  console.log("Installing external dependencies for the project...");
  await promisify(exec)(`
    cd ${WRAPPER_APP_OUTPUT_DIR}
    npm install
    npx cap sync ${platform}
  `);

  console.log(`Zipping project to ${chalk.blue(WRAPPER_APP_OUTPUT_ZIP)}...`);
  await zip(WRAPPER_APP_OUTPUT_DIR, WRAPPER_APP_OUTPUT_ZIP);

  console.log("Project ready!");
}

function zip(root, destination) {
  const job = archiver("zip", { zlib: { level: 9 } });
  const destinationStream = fs.createWriteStream(destination);

  return new Promise((resolve, reject) => {
    job.directory(root, false);
    job.pipe(destinationStream);

    destinationStream.on("close", resolve);

    job.on("error", reject);
    destinationStream.on("error", reject);

    job.finalize();
  });
}

function resolveTemplateArguments(
  platform,
  entryUrl,
  {
    appId,
    appName,
    navigationUrl,
    additionalDomains,
    smartDialerConfig = DEFAULT_SMART_DIALER_CONFIG,
  },
) {
  const result = {
    platform,
    entryUrl,
    entryDomain: new URL(entryUrl).hostname,
  };

  if (!appId) {
    // Infer an app ID from the entry domain by reversing it (e.g. `www.example.com` becomes `com.example.www`)
    // It must be lower case, and hyphens are not allowed.
    result.appId = result.entryDomain.replaceAll("-", "")
      .toLocaleLowerCase().split(".").reverse().join(".");
  }

  if (!appName) {
    // Infer an app name from the base entry domain part by title casing the root domain:
    // (e.g. `www.my-example-app.com` becomes "My Example App")
    const rootDomain = result.entryDomain.split(".").reverse()[1];

    result.appName = rootDomain.toLocaleLowerCase().replaceAll(
      /\w[a-z0-9]*[-_]*/g,
      (match) => {
        match = match.replace(/[-_]+/, " ");

        return match.charAt(0).toUpperCase() + match.slice(1).toLowerCase();
      },
    );
  }

  if (navigationUrl) {
    result.entryUrl = navigationUrl;
    result.entryDomain = new URL(navigationUrl).hostname;
  }

  if (typeof additionalDomains === "string") {
    result.additionalDomains = additionalDomains.split(/,\s*/);
    result.domainList = [result.entryDomain, ...result.additionalDomains].join(
      "\\n",
    );
  } else if (typeof additionalDomains === "object") {
    result.additionalDomains = additionalDomains;
    result.domainList = [result.entryDomain, ...result.additionalDomains].join(
      "\\n",
    );
  } else {
    result.domainList = [result.entryDomain];
  }

  result.smartDialerConfig = Buffer.from(smartDialerConfig).toString("base64");

  return result;
}

if (import.meta.url.endsWith(process.argv[1])) {
  const args = minimist(process.argv.slice(2));

  if (!args.platform) {
    throw new Error(`Parameter \`--platform\` not provided.`);
  }

  if (!args.entryUrl) {
    throw new Error(`Parameter \`--entryUrl\` not provided.`);
  }

  main({
    ...args,
    additionalDomains: args.additionalDomains?.split(/,\s*/) ?? [],
  })
    .catch(console.error);
}
