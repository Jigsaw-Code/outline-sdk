// WIP

import startNavigationProxy from "./start_proxy.mjs";
import buildAppTemplate from "./build_app.mjs";

(async function(){
  const platform = process.argv[2];

  const proxyLocations = await startNavigationProxy();

  // TODO: run build_mobileproxy

  await buildAppTemplate(proxyLocations);

  // TODO: cd output/wrapper_app, npm ci, npx cap sync ${platform}, npx cap open ${platform}
})();