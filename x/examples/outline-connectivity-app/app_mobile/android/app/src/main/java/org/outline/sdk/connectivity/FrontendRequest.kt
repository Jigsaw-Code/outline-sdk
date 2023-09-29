package org.outline.sdk.connectivity

import kotlinx.serialization.Serializable

@Serializable
data class FrontendRequest(
    val resourceName: String,
    val parameters: String
)
