package org.outline.sdk.connectivity

import kotlinx.serialization.Serializable

@Serializable
data class FrontendResponse(
    val body: String,
    val error: String
)