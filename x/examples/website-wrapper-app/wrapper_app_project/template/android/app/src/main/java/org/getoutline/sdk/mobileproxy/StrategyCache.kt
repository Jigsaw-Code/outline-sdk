package org.getoutline.sdk.mobileproxy

import android.content.SharedPreferences
import androidx.core.content.edit

import mobileproxy.StrategyCache

class SharedPreferencesStrategyCache(private val preferences: SharedPreferences) : StrategyCache {
    override fun get(key: String?): String? {
        return try {
            preferences.getString(key, null)
        } catch (_: Exception) {
            null
        }
    }

    override fun put(key: String?, value: String?) {
        try {
            preferences.edit {
                if (value.isNullOrEmpty()) {
                    remove(key)
                } else {
                    putString(key, value)
                }
            }
        } catch (_: Exception) {
        }
    }
}
