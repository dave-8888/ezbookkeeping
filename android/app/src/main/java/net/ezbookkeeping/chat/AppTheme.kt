package net.ezbookkeeping.chat

import android.app.Activity
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.core.view.WindowCompat

val LocalNight = staticCompositionLocalOf { false }
@Composable fun BookkeepingTheme(content: @Composable () -> Unit) {
    val night = isSystemInDarkTheme()
    val colors = if (night) darkColorScheme(
        primary = Color(0xFF64ACFF), onPrimary = Color.Black,
        primaryContainer = Color(0xFF193C62), onPrimaryContainer = Color(0xFFE0EDFF),
        background = Color(0xFF10151B), surface = Color(0xFF22272E),
        surfaceContainerHigh = Color(0xFF282D35), onSurface = Color(0xFFF5F5F7),
        onBackground = Color(0xFFF5F5F7), onSurfaceVariant = Color(0xFFADB4BF),
        outline = Color(0xFF626975), outlineVariant = Color(0xFF383E47)
    ) else lightColorScheme(
        primary = Color(0xFF006BDC), onPrimary = Color.White,
        primaryContainer = Color(0xFFE0EDFF), onPrimaryContainer = Color(0xFF172C46),
        background = Color(0xFFF2F2F7), surface = Color.White,
        surfaceContainerHigh = Color.White, onSurface = Color(0xFF171719),
        onBackground = Color(0xFF171719), onSurfaceVariant = Color(0xFF686871),
        outline = Color(0xFF8E8E93), outlineVariant = Color(0xFFDDDEE4)
    )
    val view = LocalView.current
    SideEffect {
        (view.context as? Activity)?.window?.let {
            WindowCompat.getInsetsController(it, view).apply {
                isAppearanceLightStatusBars = !night
                isAppearanceLightNavigationBars = !night
            }
        }
    }
    CompositionLocalProvider(LocalNight provides night) {
        MaterialTheme(colorScheme = colors, shapes = Shapes(
            small = RoundedCornerShape(10.dp), medium = RoundedCornerShape(14.dp),
            large = RoundedCornerShape(18.dp), extraLarge = RoundedCornerShape(24.dp)
        ), typography = Typography(
            headlineMedium = TextStyle(fontSize = 32.sp, lineHeight = 38.sp, fontWeight = FontWeight.Bold),
            headlineSmall = TextStyle(fontSize = 26.sp, lineHeight = 32.sp, fontWeight = FontWeight.Bold),
            titleLarge = TextStyle(fontSize = 22.sp, lineHeight = 28.sp, fontWeight = FontWeight.SemiBold),
            bodyLarge = TextStyle(fontSize = 17.sp, lineHeight = 25.sp),
            bodyMedium = TextStyle(fontSize = 15.sp, lineHeight = 22.sp),
            bodySmall = TextStyle(fontSize = 13.sp, lineHeight = 19.sp),
            labelLarge = TextStyle(fontSize = 17.sp, lineHeight = 22.sp, fontWeight = FontWeight.Medium)
        ), content = content)
    }
}

/** Shared quiet field surface, including dialogs and transaction editing. */
@Composable fun AppField(
    value: String, onValueChange: (String) -> Unit, modifier: Modifier = Modifier,
    label: (@Composable () -> Unit)? = null, placeholder: (@Composable () -> Unit)? = null,
    supportingText: (@Composable () -> Unit)? = null, singleLine: Boolean = false,
    enabled: Boolean = true, maxLines: Int = Int.MAX_VALUE,
    keyboardOptions: KeyboardOptions = KeyboardOptions.Default,
    visualTransformation: VisualTransformation = VisualTransformation.None,
    shape: androidx.compose.ui.graphics.Shape = RoundedCornerShape(12.dp)
) {
    OutlinedTextField(value, onValueChange, modifier, enabled = enabled, label = label,
        placeholder = placeholder, supportingText = supportingText, singleLine = singleLine,
        maxLines = maxLines, keyboardOptions = keyboardOptions, visualTransformation = visualTransformation,
        shape = shape, colors = OutlinedTextFieldDefaults.colors(
            unfocusedContainerColor = MaterialTheme.colorScheme.surface,
            focusedContainerColor = MaterialTheme.colorScheme.surface,
            disabledContainerColor = MaterialTheme.colorScheme.surface,
            unfocusedBorderColor = MaterialTheme.colorScheme.outlineVariant
        ))
}
