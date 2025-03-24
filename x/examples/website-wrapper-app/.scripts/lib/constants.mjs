import path from "node:path";

export const OUTPUT_DIR = path.join(process.cwd(), "output");

export const NAVIGATION_PROXY_DIR = path.join(process.cwd(), "navigation_proxy");

export const WRAPPER_APP_TEMPLATE_DIR = path.join(
  process.cwd(),
  "wrapper_app_template",
);
export const WRAPPER_APP_OUTPUT_DIR = path.join(OUTPUT_DIR, "wrapper_app");
export const WRAPPER_APP_OUTPUT_ZIP = path.join(OUTPUT_DIR, "wrapper_app.zip");

export const SDK_MOBILEPROXY_OUTPUT_DIR = path.join(OUTPUT_DIR, "mobileproxy");
export const WRAPPER_APP_OUTPUT_SDK_MOBILEPROXY_DIR = path.join(
  WRAPPER_APP_OUTPUT_DIR,
  "mobileproxy",
);

export const DEFAULT_SMART_DIALER_CONFIG = {
  dns: [{
    https: { name: "9.9.9.9" },
  }],
  tls: ["", "split:1", "split:2", "tlsfrag:1"],
};
