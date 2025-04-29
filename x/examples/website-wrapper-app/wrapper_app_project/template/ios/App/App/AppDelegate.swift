import Foundation
import Capacitor
import Mobileproxy
import UIKit

@UIApplicationMain
class AppDelegate: UIResponder, UIApplicationDelegate {

    var window: UIWindow?

    private var proxy: MobileproxyProxy? = nil

    func application(
        _ application: UIApplication,
        didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?
    ) -> Bool {

        self.resetViewController()

        return true
    }

    func applicationDidBecomeActive(_ application: UIApplication) {
        if !self.tryProxy() {
            // try one more time
            self.tryProxy()
        }
    }

    func applicationWillResignActive(_ application: UIApplication) {
        if let proxy = self.proxy {
            proxy.stop(3000)
            self.proxy = nil
        }
    }

    func application(
        _ app: UIApplication, open url: URL, options: [UIApplication.OpenURLOptionsKey: Any] = [:]
    ) -> Bool {
        // Called when the app was launched with a url. Feel free to add additional processing here,
        // but if you want the App API to support tracking app url opens, make sure to keep this call
        return ApplicationDelegateProxy.shared.application(app, open: url, options: options)
    }

    func application(
        _ application: UIApplication, continue userActivity: NSUserActivity,
        restorationHandler: @escaping ([UIUserActivityRestoring]?) -> Void
    ) -> Bool {
        // Called when the app was launched with an activity, including Universal Links.
        // Feel free to add additional processing here, but if you want the App API to support
        // tracking app url opens, make sure to keep this call
        return ApplicationDelegateProxy.shared.application(
            application, continue: userActivity, restorationHandler: restorationHandler)
    }

    @discardableResult
    private func tryProxy() -> Bool {
        guard let smartDialerConfig = Data(base64Encoded: Config.smartDialer) else {
            return false
        }

        var error: NSError?
        if let dialer = MobileproxyNewSmartStreamDialer(
            MobileproxyNewListFromLines(Config.domainList),
            String(data: smartDialerConfig, encoding: .utf8),
            MobileproxyNewStderrLogWriter(),
            &error
        ) {
            if error != nil {
                return false
            }

            self.proxy = MobileproxyRunProxy(
                "127.0.0.1:0",
                dialer,
                &error
            )

            Config.proxyPort = String(self.proxy?.port() ?? 0)
            
            self.resetViewController()

            if error != nil {
                return false
            }
        }

        return true
    }

    private func resetViewController() {
        self.window?.rootViewController = OutlineBridgeViewController()
    }
}
