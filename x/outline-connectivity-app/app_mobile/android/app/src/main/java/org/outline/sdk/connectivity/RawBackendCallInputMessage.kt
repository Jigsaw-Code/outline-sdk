package org.outline.sdk.connectivity

import kotlinx.serialization.Serializable;

@Serializable
data class RawBackendCallInputMessage(
    val method: String,
    val input: String
)
