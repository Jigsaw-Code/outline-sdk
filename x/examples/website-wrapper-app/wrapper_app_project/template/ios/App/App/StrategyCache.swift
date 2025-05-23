import Foundation
import Mobileproxy

class UserDefaultsStrategyCache : NSObject, MobileproxyStrategyCacheProtocol {
    private let userDefaults: UserDefaults

    init(userDefaults: UserDefaults = .standard) {
        self.userDefaults = userDefaults
        super.init()
    }

    func get(_ key: String?) -> String {
        guard let nonNilKey = key else { return "" }
        return self.userDefaults.string(forKey: nonNilKey) ?? ""
    }

    func put(_ key: String?, value: String?) {
        guard let nonNilKey = key else { return }
        if let nonNilVal = value, !nonNilVal.isEmpty {
            userDefaults.set(nonNilVal, forKey: nonNilKey)
        } else {
            userDefaults.removeObject(forKey: nonNilKey)
        }
    }
}
