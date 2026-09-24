<template>
    <div class="layout-wrapper layout-nav-type-vertical layout-navbar-static layout-footer-static layout-content-width-fluid"
         :class="{ 'layout-overlay-nav': mdAndDown }">
        <div class="layout-vertical-nav" :class="{'visible': showVerticalOverlayMenu, 'scrolled': isVerticalNavScrolled, 'overlay-nav': mdAndDown}">
            <div class="nav-header">
                <router-link to="/" class="app-logo d-flex align-center gap-x-3 app-title-wrapper" @click="refreshOverviewPage">
                    <div class="d-flex">
                        <img alt="logo" class="main-logo" :src="APPLICATION_LOGO_PATH" />
                    </div>
                    <h1 class="font-weight-medium text-xl">{{ tt('global.app.title') }}</h1>
                </router-link>
                <div class="nav-header-actions">
                    <v-btn color="primary" variant="text" size="30"
                           :icon="true" :aria-label="showAmountGlobally ? tt('Hide Amount') : tt('Show Amount')"
                           @click="showAmountGlobally = !showAmountGlobally">
                        <v-icon :icon="showAmountGlobally ? mdiEyeOffOutline : mdiEyeOutline" size="18" />
                        <v-tooltip activator="parent">{{ showAmountGlobally ? tt('Hide Amount') : tt('Show Amount') }}</v-tooltip>
                    </v-btn>
                    <v-btn color="primary" variant="text" size="30"
                           :icon="true" :aria-label="tt('Theme')" @click="(currentTheme === 'light' ? currentTheme = 'dark' : (currentTheme === 'dark' ? currentTheme = 'auto' : currentTheme = 'light'))">
                        <v-icon :icon="(currentTheme === 'light' ? mdiWeatherSunny : (currentTheme === 'dark' ? mdiWeatherNight : mdiThemeLightDark))" size="18" />
                    </v-btn>
                    <v-btn :icon="true" size="30" variant="text" color="primary" :aria-label="tt('User Settings')">
                        <v-avatar size="26" variant="tonal" :color="currentUserAvatar ? 'rgba(0,0,0,0)' : 'primary'">
                            <v-img :src="currentUserAvatar" v-if="currentUserAvatar">
                                <template #placeholder>
                                    <div class="d-flex align-center justify-center fill-height bg-light-primary">
                                        <v-icon color="primary" :icon="mdiAccount" size="18"/>
                                    </div>
                                </template>
                            </v-img>
                            <v-icon :icon="mdiAccount" size="18" v-else-if="!currentUserAvatar"/>
                        </v-avatar>
                        <v-menu activator="parent" width="230" location="bottom end" offset="14px">
                            <v-list>
                                <v-list-item>
                                    <template #prepend>
                                        <v-list-item-action>
                                            <v-avatar variant="tonal"
                                                      :color="currentUserAvatar ? 'rgba(0,0,0,0)' : 'primary'">
                                                <v-img :src="currentUserAvatar" v-if="currentUserAvatar">
                                                    <template #placeholder>
                                                        <div class="d-flex align-center justify-center fill-height bg-light-primary">
                                                            <v-icon color="primary" :icon="mdiAccount"/>
                                                        </div>
                                                    </template>
                                                </v-img>
                                                <v-icon :icon="mdiAccount" v-else-if="!currentUserAvatar"/>
                                            </v-avatar>
                                        </v-list-item-action>
                                    </template>
                                    <v-list-item-title class="ms-2">
                                        {{ currentNickName }}
                                    </v-list-item-title>
                                </v-list-item>
                                <v-divider class="my-2"/>
                                <v-list-item :prepend-icon="mdiAccountCogOutline"
                                             :title="tt('User Settings')"
                                             to="/user/settings"></v-list-item>
                                <v-list-item :prepend-icon="mdiCogOutline"
                                             :title="tt('Application Settings')"
                                             to="/app/settings"></v-list-item>
                                <v-divider class="my-2"/>
                                <v-list-item :prepend-icon="mdiLockOutline"
                                             :title="tt('Lock Application')"
                                             v-if="isEnableApplicationLock"
                                             @click="lock"></v-list-item>
                                <v-list-item :disabled="logouting"
                                             :prepend-icon="mdiLogout"
                                             :title="tt('Log Out')"
                                             @click="logout"></v-list-item>
                            </v-list>
                        </v-menu>
                    </v-btn>
                </div>
            </div>
            <perfect-scrollbar
                tag="ul" class="nav-items"
                :options="{ wheelPropagation: false }"
                @ps-scroll-y="handleNavScroll"
            >
                <li class="nav-link home-link">
                    <router-link to="/" @click="refreshOverviewPage">
                        <v-icon class="nav-item-icon" :icon="mdiHomeOutline"/>
                        <span class="nav-item-title">{{ tt('Overview') }}</span>
                    </router-link>
                </li>
                <li class="nav-section-title">
                    <div class="title-wrapper">
                        <span class="title-text">{{ tt('Transaction Data') }}</span>
                    </div>
                </li>
                <li class="nav-link">
                    <router-link to="/transaction/list?pageType=0&dateType=7">
                        <v-icon class="nav-item-icon" :icon="mdiListBoxOutline"/>
                        <span class="nav-item-title d-inline-block">{{ tt('Transaction Details') }}</span>
                        <v-btn density="compact" color="secondary" variant="text" size="22"
                               class="ms-1" :icon="true" v-if="showAddTransactionButtonInDesktopNavbar"
                               @click="showAddDialogInTransactionListPage">
                            <v-icon :icon="mdiPlusCircle" size="22" />
                            <v-tooltip activator="parent">{{ tt('Add Transaction') }}</v-tooltip>
                        </v-btn>
                    </router-link>
                </li>
                <li class="nav-link">
                    <router-link to="/statistics/transaction">
                        <v-icon class="nav-item-icon" :icon="mdiChartPieOutline"/>
                        <span class="nav-item-title">{{ tt('Statistics & Analysis') }}</span>
                    </router-link>
                </li>
                <li class="nav-link">
                    <router-link to="/social-gifts">
                        <v-icon class="nav-item-icon" :icon="mdiGiftOutline"/>
                        <span class="nav-item-title">{{ tt('Social Gifts') }}</span>
                    </router-link>
                </li>
                <li class="nav-link">
                    <router-link to="/insights/explore">
                        <v-icon class="nav-item-icon" :icon="mdiCompassOutline"/>
                        <span class="nav-item-title">{{ tt('Insights & Explore') }}</span>
                    </router-link>
                </li>
                <li class="nav-section-title">
                    <div class="title-wrapper">
                        <span class="title-text">{{ tt('Basis Data') }}</span>
                    </div>
                </li>
                <li class="nav-link">
                    <router-link to="/account/list">
                        <v-icon class="nav-item-icon" :icon="mdiCreditCardOutline"/>
                        <span class="nav-item-title">{{ tt('Accounts') }}</span>
                    </router-link>
                </li>
                <li class="nav-link">
                    <router-link to="/account/quick_balance_modify">
                        <v-icon class="nav-item-icon" :icon="mdiCashEdit"/>
                        <span class="nav-item-title">{{ tt('Quick Account Balance Modification') }}</span>
                    </router-link>
                </li>
                <li class="nav-link">
                    <router-link to="/category/list">
                        <v-icon class="nav-item-icon" :icon="mdiViewDashboardOutline"/>
                        <span class="nav-item-title">{{ tt('Transaction Categories') }}</span>
                    </router-link>
                </li>
                <li class="nav-link">
                    <router-link to="/tag/list">
                        <v-icon class="nav-item-icon" :icon="mdiTagOutline"/>
                        <span class="nav-item-title">{{ tt('Transaction Tags') }}</span>
                    </router-link>
                </li>
                <li class="nav-link">
                    <router-link to="/template/list">
                        <v-icon class="nav-item-icon" :icon="mdiClipboardTextOutline"/>
                        <span class="nav-item-title">{{ tt('Transaction Templates') }}</span>
                    </router-link>
                </li>
                <li class="nav-link" v-if="isUserScheduledTransactionEnabled()">
                    <router-link to="/schedule/list">
                        <v-icon class="nav-item-icon" :icon="mdiClipboardTextClockOutline"/>
                        <span class="nav-item-title">{{ tt('Scheduled Transactions') }}</span>
                    </router-link>
                </li>
                <li class="nav-link">
                    <router-link to="/asset-planning">
                        <v-icon class="nav-item-icon" :icon="mdiTarget"/>
                        <span class="nav-item-title">{{ tt('Asset Planning') }}</span>
                    </router-link>
                </li>
                <li class="nav-section-title">
                    <div class="title-wrapper">
                        <span class="title-text">{{ tt('Miscellaneous') }}</span>
                    </div>
                </li>
                <li class="nav-link">
                    <router-link to="/exchange_rates">
                        <v-icon class="nav-item-icon" :icon="mdiSwapHorizontal"/>
                        <span class="nav-item-title">{{ tt('Exchange Rates Data') }}</span>
                    </router-link>
                </li>
                <li class="nav-link">
                    <a href="javascript:void(0);" @click="showMobileQrCode = true">
                        <v-icon class="nav-item-icon" :icon="mdiCellphone"/>
                        <span class="nav-item-title">{{ tt('Use on Mobile Device') }}</span>
                    </a>
                </li>
                <li class="nav-link">
                    <router-link to="/about">
                        <v-icon class="nav-item-icon" :icon="mdiInformationOutline"/>
                        <span class="nav-item-title">{{ tt('About') }}</span>
                    </router-link>
                </li>
            </perfect-scrollbar>
        </div>

        <v-btn v-if="mdAndDown && !showVerticalOverlayMenu" class="compact-nav-toggle"
               color="primary" size="40" :icon="true" :aria-label="tt('Navigation Bar')"
               @click="showVerticalOverlayMenu = true">
            <v-icon :icon="mdiMenu" size="22" />
        </v-btn>

        <div class="layout-content-wrapper">
            <div class="layout-page-content">
                <div class="page-content-container">
                    <router-view :key="currentRoutePath" />
                </div>
            </div>
        </div>

        <switch-to-mobile-dialog v-model:show="showMobileQrCode" />

        <div class="layout-overlay" :class="{ 'visible': showVerticalOverlayMenu }" @click="showVerticalOverlayMenu = false"></div>

        <v-overlay class="justify-center align-center" :persistent="true" v-model="showLoading">
            <v-progress-circular indeterminate></v-progress-circular>
        </v-overlay>

        <global-quick-transaction-button />

        <snack-bar ref="snackbar" />
    </div>
</template>

<script setup lang="ts">
import GlobalQuickTransactionButton from '@/components/desktop/GlobalQuickTransactionButton.vue';
import SnackBar from '@/components/desktop/SnackBar.vue';

import { ref, computed, useTemplateRef } from 'vue';

import { useDisplay, useTheme } from 'vuetify';
import { useRoute, useRouter } from 'vue-router';

import { useI18n } from '@/locales/helpers.ts';

import { useRootStore } from '@/stores/index.ts';
import { useSettingsStore } from '@/stores/setting.ts';
import { useUserStore } from '@/stores/user.ts';
import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import { APPLICATION_LOGO_PATH } from '@/consts/asset.ts';
import { ThemeType } from '@/core/theme.ts';
import { isUserScheduledTransactionEnabled } from '@/lib/server_settings.ts';
import { getSystemTheme, setExpenseAndIncomeAmountColor } from '@/lib/ui/common.ts';

import {
    mdiMenu,
    mdiHomeOutline,
    mdiListBoxOutline,
    mdiPlusCircle,
    mdiCashEdit,
    mdiCreditCardOutline,
    mdiViewDashboardOutline,
    mdiTagOutline,
    mdiClipboardTextOutline,
    mdiClipboardTextClockOutline,
    mdiTarget,
    mdiChartPieOutline,
    mdiCompassOutline,
    mdiSwapHorizontal,
    mdiCogOutline,
    mdiCellphone,
    mdiInformationOutline,
    mdiThemeLightDark,
    mdiWeatherSunny,
    mdiWeatherNight,
    mdiEyeOutline,
    mdiEyeOffOutline,
    mdiAccount,
    mdiAccountCogOutline,
    mdiLockOutline,
    mdiLogout,
    mdiGiftOutline
} from '@mdi/js';

type SnackBarType = InstanceType<typeof SnackBar>;

const display = useDisplay();
const theme = useTheme();
const route = useRoute();
const router = useRouter();

const { tt, initLocale } = useI18n();

const rootStore = useRootStore();
const settingsStore = useSettingsStore();
const userStore = useUserStore();
const desktopPageStore = useDesktopPageStore();

const snackbar = useTemplateRef<SnackBarType>('snackbar');

const logouting = ref<boolean>(false);
const isVerticalNavScrolled = ref<boolean>(false);
const showVerticalOverlayMenu = ref<boolean>(false);
const showLoading = ref<boolean>(false);
const showMobileQrCode = ref<boolean>(false);

const mdAndDown = computed<boolean>(() => display.mdAndDown.value);
const currentRoutePath = computed<string>(() => route.path);

const currentNickName = computed<string>(() => userStore.currentUserNickname || tt('User'));
const currentUserAvatar = computed<string | null>(() => userStore.getUserAvatarUrl(userStore.currentUserBasicInfo, true));

const currentTheme = computed<string>({
    get: () => {
        return settingsStore.appSettings.theme;
    },
    set: (value: string) => {
        if (value !== settingsStore.appSettings.theme) {
            settingsStore.setTheme(value);

            if (value === ThemeType.Light || value === ThemeType.Dark) {
                theme.change(value);
            } else {
                theme.change(getSystemTheme());
            }
        }
    }
});

const showAddTransactionButtonInDesktopNavbar = computed<boolean>(() => settingsStore.appSettings.showAddTransactionButtonInDesktopNavbar);
const isEnableApplicationLock = computed<boolean>(() => settingsStore.appSettings.applicationLock);
const showAmountGlobally = computed<boolean>({
    get: () => settingsStore.appSettings.showAmountInHomePage &&
        settingsStore.appSettings.showAccountBalance &&
        settingsStore.appSettings.showTotalAmountInTransactionListPage,
    set: (value: boolean) => {
        settingsStore.setShowAmountInHomePage(value);
        settingsStore.setShowAccountBalance(value);
        settingsStore.setShowTotalAmountInTransactionListPage(value);
    }
});

function handleNavScroll(e: Event): void {
    isVerticalNavScrolled.value = (e.target as HTMLElement).scrollTop > 0;
}

function lock(): void {
    rootStore.lock();
    router.replace('/unlock');
}

function logout(): void {
    logouting.value = true;
    showLoading.value = true;

    rootStore.logout().then(() => {
        logouting.value = false;
        showLoading.value = false;

        settingsStore.clearAppSettings();

        const localeDefaultSettings = initLocale(userStore.currentUserLanguage, settingsStore.appSettings.timeZone);
        settingsStore.updateLocalizedDefaultSettings(localeDefaultSettings);

        setExpenseAndIncomeAmountColor(userStore.currentUserExpenseAmountColor, userStore.currentUserIncomeAmountColor);

        router.replace('/login');
    }).catch(error => {
        logouting.value = false;
        showLoading.value = false;

        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

function showAddDialogInTransactionListPage(): void {
    desktopPageStore.setShowAddTransactionDialogInTransactionList();
}

function refreshOverviewPage(): void {
    desktopPageStore.requestOverviewRefresh();
}
</script>

<style>
.main-logo {
    width: 1.75rem;
    height: 1.75rem;
}

.nav-link.home-link > a:not(.router-link-exact-active):hover::before {
    opacity: calc(var(--v-hover-opacity)* var(--v-theme-overlay-multiplier));
}
</style>

<style scoped>
.layout-vertical-nav .nav-header {
    padding: 16px;
    gap: 8px;
    flex-shrink: 0;
}

.layout-vertical-nav .nav-header .app-logo {
    min-width: 0;
    flex-shrink: 1;
    gap: 8px !important;
}

.nav-header .app-logo h1 {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 18px !important;
}

.nav-header-actions {
    display: flex;
    align-items: center;
    gap: 2px;
    flex-shrink: 0;
    margin-inline-start: auto;
}

.compact-nav-toggle {
    position: fixed;
    inset-inline-start: 16px;
    bottom: calc(16px + env(safe-area-inset-bottom));
    z-index: 12;
}
</style>
