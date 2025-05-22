package com.example.mobileproxy.test

import androidx.appcompat.app.AppCompatActivity
import android.os.Bundle
import android.widget.TextView
// It's good practice to import R specifically if you have name clashes,
// but for a simple app, com.example.mobileproxy.test.R is fine.
// import com.example.mobileproxy.test.R 

class MainActivity : AppCompatActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        // Basic "Hello World" UI
        // setContentView(R.layout.activity_main) // Assuming you'll create a layout file later
        
        // For now, let's just set a TextView programmatically to avoid needing a layout XML yet
        val textView = TextView(this).apply {
            text = "Hello, MobileProxy Test!"
            textSize = 20f
            gravity = android.view.Gravity.CENTER
        }
        setContentView(textView)

        // Example of how you might interact with the proxy (conceptual)
        // This would require the mobileproxy library to be initialized and its API available
        // For now, this is just a placeholder comment
        /*
        try {
            // Assuming 'Mobileproxy' is a class from your AAR
            // And it has a function to start the proxy or get its status
            // val proxyStatus = Mobileproxy.getStatus()
            // textView.append("\nProxy Status: $proxyStatus")
        } catch (e: Exception) {
            textView.append("\nError accessing proxy: ${e.message}")
        }
        */
    }
}
