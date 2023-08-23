package org.outline.sdk.connectivity

import kotlinx.serialization.Serializable

@Serializable
data class RawBackendCallOutputMessage(
    val result: String,
    val errors: List<String>
)
