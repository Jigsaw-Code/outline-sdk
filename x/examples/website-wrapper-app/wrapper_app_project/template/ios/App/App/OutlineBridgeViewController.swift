import UIKit
import Capacitor

class OutlineBridgeViewController: CAPBridgeViewController {
    override func webView(with frame: CGRect, configuration: WKWebViewConfiguration) -> WKWebView {
        if #available(iOS 17.0, *) {
            let endpoint = NWEndpoint.hostPort(
                host: NWEndpoint.Host(Config.proxyHost),
                port: NWEndpoint.Port(Config.proxyPort)!
            )
            let proxyConfig = ProxyConfiguration.init(httpCONNECTProxy: endpoint)

            let websiteDataStore = WKWebsiteDataStore.default()
            websiteDataStore.proxyConfigurations = [proxyConfig]

            configuration.websiteDataStore = websiteDataStore
        } else {
            // TODO: use scheme handler
        }

        return super.webView(with: frame, configuration: configuration)
    }
    
    override open func viewWillLayoutSubviews() {
        super.viewWillLayoutSubviews()

        guard let webView = self.webView else { return }
        
        if let safeAreaInsets = self.view.window?.safeAreaInsets {
            webView.frame.origin = CGPoint(x: safeAreaInsets.left, y: safeAreaInsets.top)
            webView.frame.size = CGSize(
                width: UIScreen.main.bounds.width - safeAreaInsets.left - safeAreaInsets.right,
                height: UIScreen.main.bounds.height - safeAreaInsets.top - safeAreaInsets.bottom
            )
        }
    }
}
