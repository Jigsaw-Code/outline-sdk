import { CapacitorConfig } from '@capacitor/cli';

let config: CapacitorConfig = {
  appId: "org.getoutline.sdk.example.connectivity",
  appName: "@outline_sdk_connectivity_demo/app_mobile",
  webDir: "output/frontend",
  server: {
    url: "http://10.0.2.2:3000"
  }
}

switch (process.env.CAPACITOR_PLATFORM) {
  case "android":
    config.server =  {
      url: "http://10.0.2.2:3000",
      cleartext: true
    };
    break;
  case "ios":
  default:
    config.server =  {
      url: "http://localhost:3000"
    };
    break;
}

export default config;
