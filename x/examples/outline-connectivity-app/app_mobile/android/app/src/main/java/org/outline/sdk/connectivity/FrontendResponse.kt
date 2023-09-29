package org.getoutline.sdk.example.connectivity

import kotlinx.serialization.Serializable

@Serializable
data class FrontendResponse(
    val body: String,
    val error: String
)