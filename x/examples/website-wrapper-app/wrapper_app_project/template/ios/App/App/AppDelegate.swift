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
        var dialerError: NSError?
        if let dialer = MobileproxyNewSmartStreamDialer(
            MobileproxyNewListFromLines(Config.domainList),
            Config.smartDialer,
            MobileproxyNewStderrLogWriter(),
            &dialerError
        ) {
            if let error = dialerError {
                self.showError(error)
                return
            }

            var proxyError: NSError?
            self.proxy = MobileproxyRunProxy(
                "127.0.0.1:8080",
                dialer,
                &proxyError
            )

            if let error = proxyError {
                self.showError(error)

                self.proxy = nil
                return
            }
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

    private func showError(error: NSError) {
        guard let rootViewController = self.window?.rootViewController else {
            print("Could not find root view controller to present error.")
            return
        }

        let dialog = UIAlertController(title: "Error", message: error.localizedDescription, preferredStyle: .alert)

        dialog.addAction(UIAlertAction(title: "OK", style: .deault))

        rootViewController.present(alert, animated: false)
    }
}
