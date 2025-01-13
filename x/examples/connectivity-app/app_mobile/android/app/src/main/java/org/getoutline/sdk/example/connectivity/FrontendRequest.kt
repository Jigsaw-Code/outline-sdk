package org.getoutline.sdk.example.connectivity

import kotlinx.serialization.Serializable

@Serializable
data class FrontendRequest(
    val resourceName: String,
    val parameters: String
)
