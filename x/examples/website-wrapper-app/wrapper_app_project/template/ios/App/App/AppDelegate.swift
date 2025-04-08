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

        self.window?.rootViewController = OutlineBridgeViewController()

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
        var dialerError: NSError?
        if let dialer = MobileproxyNewSmartStreamDialer(
            MobileproxyNewListFromLines(Config.domainList),
            Config.smartDialer,
            MobileproxyNewStderrLogWriter(),
            &dialerError
        ) {
            if let _ = dialerError {
                return false
            }

            var proxyError: NSError?
            self.proxy = MobileproxyRunProxy(
                "127.0.0.1:0",
                dialer,
                &proxyError
            )
            
            Config.proxyPort = String(self.proxy?.port() ?? 0)

            if let _ = proxyError {
                return false
            }
        }

        return true
    }
}
