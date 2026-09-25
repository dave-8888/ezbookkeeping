<template>
    <v-row class="match-height">
        <v-col cols="12">
            <v-card class="transaction-workspace" variant="flat">
                <v-layout>
                    <v-main>
                        <v-window class="d-flex flex-grow-1 disable-tab-transition w-100-window-container" v-model="activeTab">
                            <v-window-item value="transactionPage">
                                <v-card class="transaction-content" variant="flat">
                                    <template #title>
                                        <div class="transaction-toolbar">
                                            <v-btn-toggle class="transaction-view-switch" mandatory divided
                                                          :disabled="loading" v-model="queryPageType">
                                                <v-btn v-for="item in desktopPageTypes" :key="item.type"
                                                       :value="item.type" size="small">{{ tt(item.name) }}</v-btn>
                                            </v-btn-toggle>
                                            <div class="transaction-keyword-filter">
                                                <v-text-field density="compact" variant="solo" flat hide-details
                                                              :disabled="loading" :prepend-inner-icon="mdiMagnify"
                                                              :append-inner-icon="searchKeyword !== query.keyword ? mdiCheck : undefined"
                                                              :placeholder="tt('Search transaction description')"
                                                              :aria-label="tt('Search transaction description')"
                                                              v-model="searchKeyword"
                                                              @click:append-inner="changeKeywordFilter(searchKeyword)"
                                                              @keyup.enter="changeKeywordFilter(searchKeyword)" />
                                            </div>
                                            <v-menu :close-on-content-click="false" location="bottom end">
                                                <template #activator="{ props }">
                                                    <v-btn v-bind="props" variant="text" :color="query.type ? 'primary' : 'default'"
                                                           :prepend-icon="mdiTuneVariant">{{ tt('Filters') }}</v-btn>
                                                </template>
                                                <v-card class="pa-5 transaction-filter-popover" width="300" rounded="lg">
                                                    <div class="mb-4">
                                                        <span class="text-subtitle-2">{{ tt('Transaction Type') }}</span>
                                                        <v-select
                                                            :aria-label="tt('Transaction Type')"
                                                            item-title="displayName"
                                                            item-value="type"
                                                            class="mt-2" hide-details
                                                            density="compact"
                                                            :disabled="loading"
                                                            :items="[
                                                                { displayName: tt('All Types'), type: 0 },
                                                                { displayName: tt('Modify Balance'), type: 1 },
                                                                { displayName: tt('Income'), type: 2 },
                                                                { displayName: tt('Expense'), type: 3 },
                                                                { displayName: tt('Transfer'), type: 4 }
                                                            ]"
                                                            v-model="queryType"
                                                        />
                                                    </div>

                                                </v-card>
                                            </v-menu>
                                            <v-btn class="transaction-add-button" color="primary" variant="flat" :prepend-icon="mdiPlus"
                                                   :disabled="loading || !canAddTransaction" @click="add()">
                                                {{ tt('Add') }}
                                                <v-menu activator="parent" :open-on-hover="true" v-if="isSpeechRecognitionSupported() || isTransactionFromAIImageRecognitionEnabled() || (allTransactionTemplates && allTransactionTemplates.length)">
                                                    <v-list>
                                                        <v-list-item key="VoiceInput"
                                                                     :title="tt('Voice Input')"
                                                                     :prepend-icon="mdiMicrophone"
                                                                     :disabled="recognizingByVoice"
                                                                     v-if="isSpeechRecognitionSupported()"
                                                                     @click="addByVoiceInput"></v-list-item>
                                                        <v-list-item key="AIImageRecognition"
                                                                     :title="tt('AI Image Recognition')"
                                                                     :prepend-icon="mdiMagicStaff"
                                                                     v-if="isTransactionFromAIImageRecognitionEnabled()"
                                                                     @click="addByRecognizingImage"></v-list-item>
                                                        <v-list-item :key="template.id"
                                                                     :title="template.name"
                                                                     :prepend-icon="mdiTextBoxOutline"
                                                                     v-for="template in allTransactionTemplates"
                                                                     @click="add(template)"></v-list-item>
                                                    </v-list>
                                                </v-menu>
                                            </v-btn>

                                            <v-menu location="bottom end">
                                                <template #activator="{ props }">
                                                    <v-btn v-bind="props" variant="text" :icon="mdiDotsHorizontal"
                                                           size="small" :aria-label="tt('More')" :loading="loading" />
                                                </template>
                                                <v-list density="compact" rounded="lg">
                                                    <v-list-item :title="tt('Refresh')" :prepend-icon="mdiRefresh"
                                                                 :disabled="loading" @click="reload(true, false)" />
                                                    <v-list-item :title="tt('Import')" :prepend-icon="mdiImport"
                                                                 :disabled="loading" v-if="isDataImportingEnabled()" @click="importTransaction" />
                                                    <template v-if="isDataExportingEnabled()">
                                                        <v-list-item :title="tt('Export to CSV (Comma-separated values) File')"
                                                                     :disabled="loading || exportingData || !transactions.length" @click="exportTransactions('csv')" />
                                                        <v-list-item :title="tt('Export to TSV (Tab-separated values) File')"
                                                                     :disabled="loading || exportingData || !transactions.length" @click="exportTransactions('tsv')" />
                                                    </template>
                                                </v-list>
                                            </v-menu>
                                        </div>
                                    </template>

                                    <v-card-text class="pt-0">
                                        <div class="transaction-list-datetime-range d-flex align-center">
                                            <div class="transaction-date-control">
                                                <v-btn variant="text" size="x-small" :icon="mdiChevronLeft"
                                                       :aria-label="tt('Previous')" :disabled="loading || (!query.minTime && !query.maxTime)"
                                                       @click="shiftDateRange(query.minTime, query.maxTime, -1)" />
                                                <v-menu location="bottom start" max-height="400">
                                                    <template #activator="{ props }">
                                                        <v-btn v-bind="props" variant="text" :append-icon="mdiChevronDown" :disabled="loading">
                                                            {{ recentMonthDateRanges[recentDateRangeIndex]?.displayName || tt('Custom Date Range') }}
                                                        </v-btn>
                                                    </template>
                                                    <v-list density="compact" rounded="lg" :selected="[recentDateRangeIndex]">
                                                        <v-list-item v-for="(range, idx) in recentMonthDateRanges" :key="idx" :value="idx"
                                                                     :title="range.displayName" @click="changeDateFilter(range)" />
                                                    </v-list>
                                                </v-menu>
                                                <v-btn variant="text" size="x-small" :icon="mdiChevronRight"
                                                       :aria-label="tt('Next')" :disabled="loading || (!query.minTime && !query.maxTime)"
                                                       @click="shiftDateRange(query.minTime, query.maxTime, 1)" />
                                                <span class="transaction-custom-range text-caption" v-if="recentMonthDateRanges[recentDateRangeIndex]?.dateType === DateRange.Custom.type && (query.minTime || query.maxTime)">
                                                    {{ queryMinTime }} – {{ queryMaxTime }}
                                                </span>
                                            </div>
                                            <v-spacer/>
                                            <div class="transaction-totals skeleton-no-margin d-flex align-center" v-if="currentMonthTotalAmount">
                                                <span class="ms-2 text-subtitle-1">{{ queryAllFilterAccountIdsCount ? tt('Total Inflows') : tt('Total Income') }}</span>
                                                <span class="text-income ms-2" v-if="loading">
                                                    <v-skeleton-loader type="text" style="width: 60px" :loading="true"></v-skeleton-loader>
                                                </span>
                                                <span class="text-income ms-2" v-else-if="!loading">
                                                    {{ currentMonthTotalAmount.income }}
                                                </span>
                                                <span class="text-subtitle-1 ms-3">{{ queryAllFilterAccountIdsCount ? tt('Total Outflows') : tt('Total Expense') }}</span>
                                                <span class="text-expense ms-2" v-if="loading">
                                                    <v-skeleton-loader type="text" style="width: 60px" :loading="true"></v-skeleton-loader>
                                                </span>
                                                <span class="text-expense ms-2" v-else-if="!loading">
                                                    {{ currentMonthTotalAmount.expense }}
                                                </span>
                                            </div>
                                        </div>
                                    </v-card-text>

                                    <v-card-text class="transaction-calendar-container pt-0" v-if="pageType === TransactionListPageType.Calendar.type">
                                        <transaction-calendar day-has-transaction-class="font-weight-bold"
                                                              :readonly="loading" :is-dark-mode="isDarkMode"
                                                              :default-currency="defaultCurrency"
                                                              :hide-amount="!showTotalAmountInTransactionListPage"
                                                              :min-date="transactionCalendarMinDate"
                                                              :max-date="transactionCalendarMaxDate"
                                                              :dailyTotalAmounts="currentMonthTransactionData?.dailyTotalAmounts"
                                                              v-model="currentCalendarDate"></transaction-calendar>
                                    </v-card-text>

                                    <v-table class="transaction-table" :hover="!loading">
                                        <thead>
                                        <tr>
                                            <th class="transaction-table-column-description text-no-wrap">
                                                <div class="d-flex align-center ga-4">
                                                    <span>{{ pageType !== TransactionListPageType.Calendar.type ? tt('Filters') : tt('Description') }}</span>
                                                    <span v-if="pageType === analysisPageType" class="text-caption text-medium-emphasis text-truncate"
                                                          :title="`${tt('Expense rankings use the current filters and default currency')} (${defaultCurrency})`">
                                                        {{ tt('Expense rankings use the current filters and default currency') }} ({{ defaultCurrency }})
                                                    </span>
                                                </div>
                                            </th>
                                            <th class="transaction-table-column-amount text-no-wrap">
                                                <v-menu ref="amountFilterMenu" class="transaction-amount-menu"
                                                        eager location="bottom" max-height="500"
                                                        :close-on-content-click="false"
                                                        v-model="amountMenuState"
                                                        @update:model-value="scrollAmountMenuToSelectedItem">
                                                    <template #activator="{ props }">
                                                        <div class="d-flex align-center cursor-pointer"
                                                             :class="{ 'readonly': loading, 'text-primary': query.amountFilter }" v-bind="props">
                                                            <span>{{ tt('Amount') }}</span>
                                                            <v-icon :icon="mdiMenuDown" />
                                                        </div>
                                                    </template>
                                                    <v-list :selected="[query.amountFilter.split(':')[0]]">
                                                        <v-list-item key="" value="" class="text-sm" density="compact"
                                                                     :class="{ 'list-item-selected': !query.amountFilter }"
                                                                     :append-icon="(!query.amountFilter && !currentAmountFilterType ? mdiCheck : undefined)">
                                                            <v-list-item-title class="cursor-pointer"
                                                                               @click="changeAmountFilter('')">
                                                                <div class="d-flex align-center">
                                                                    <span class="text-sm ms-3">{{ tt('All') }}</span>
                                                                </div>
                                                            </v-list-item-title>
                                                        </v-list-item>
                                                        <template :key="filterType.type"
                                                                  v-for="filterType in AmountFilterType.values()">
                                                            <v-list-item class="text-sm" density="compact"
                                                                         :value="filterType.type"
                                                                         :class="{ 'list-item-selected': query.amountFilter && query.amountFilter.startsWith(`${filterType.type}:`) }"
                                                                         :append-icon="(query.amountFilter && query.amountFilter.startsWith(`${filterType.type}:`) && currentAmountFilterType !== filterType.type ? mdiCheck : undefined)">
                                                                <v-list-item-title class="cursor-pointer"
                                                                                   @click="currentAmountFilterType = filterType.type">
                                                                    <div class="d-flex align-center">
                                                                        <span class="text-sm ms-3">{{ tt(filterType.name) }}</span>
                                                                        <span class="text-sm ms-4" v-if="query.amountFilter && query.amountFilter.startsWith(`${filterType.type}:`) && currentAmountFilterType !== filterType.type">{{ queryAmount }}</span>
                                                                        <amount-input class="transaction-amount-filter-value ms-4" density="compact"
                                                                                      :currency="defaultCurrency"
                                                                                      v-model="currentAmountFilterValue1"
                                                                                      v-if="currentAmountFilterType === filterType.type"/>
                                                                        <span class="ms-2 me-2" v-if="currentAmountFilterType === filterType.type && filterType.paramCount === 2">~</span>
                                                                        <amount-input class="transaction-amount-filter-value" density="compact"
                                                                                      :currency="defaultCurrency"
                                                                                      v-model="currentAmountFilterValue2"
                                                                                      v-if="currentAmountFilterType === filterType.type && filterType.paramCount === 2"/>
                                                                        <v-btn class="ms-2" density="compact" color="primary" variant="tonal"
                                                                               @click="changeAmountFilter(filterType.type)"
                                                                               v-if="currentAmountFilterType === filterType.type">{{ tt('Apply') }}</v-btn>
                                                                    </div>
                                                                </v-list-item-title>
                                                            </v-list-item>
                                                        </template>
                                                    </v-list>
                                                </v-menu>
                                            </th>
                                            <th class="transaction-table-column-category text-no-wrap">
                                                <v-menu ref="categoryFilterMenu" class="transaction-category-menu"
                                                        eager location="bottom" max-height="500"
                                                        :disabled="query.type === 1"
                                                        :close-on-content-click="false"
                                                        v-model="categoryMenuState"
                                                        @update:model-value="scrollCategoryMenuToSelectedItem">
                                                    <template #activator="{ props }">
                                                        <div class="d-flex align-center"
                                                            :class="{ 'readonly': loading, 'cursor-pointer': query.type !== 1, 'text-primary': query.categoryIds }" v-bind="props">
                                                            <span>{{ queryCategoryName }}</span>
                                                            <v-icon :icon="mdiMenuDown" v-show="query.type !== 1" />
                                                        </div>
                                                    </template>
                                                    <v-list :selected="[queryAllSelectedFilterCategoryIds]">
                                                        <v-list-item key="" value="" class="text-sm" density="compact"
                                                                     :class="{ 'list-item-selected': !query.categoryIds }"
                                                                     :append-icon="(!query.categoryIds ? mdiCheck : undefined)">
                                                            <v-list-item-title class="cursor-pointer"
                                                                               @click="changeCategoryFilter('')">
                                                                <div class="d-flex align-center">
                                                                    <v-icon :icon="mdiViewGridOutline" />
                                                                    <span class="text-sm ms-3">{{ tt('All') }}</span>
                                                                </div>
                                                            </v-list-item-title>
                                                        </v-list-item>
                                                        <v-list-item key="multiple" value="multiple" class="text-sm" density="compact"
                                                                     :class="{ 'list-item-selected': query.categoryIds && queryAllFilterCategoryIdsCount > 1 }"
                                                                     :append-icon="(query.categoryIds && queryAllFilterCategoryIdsCount > 1 ? mdiCheck : undefined)"
                                                                     v-if="allAvailableCategoriesCount > 0">
                                                            <v-list-item-title class="cursor-pointer"
                                                                               @click="showFilterCategoryDialog = true">
                                                                <div class="d-flex align-center">
                                                                    <v-icon :icon="mdiVectorArrangeBelow" />
                                                                    <span class="text-sm ms-3">{{ tt('Multiple Categories') }}</span>
                                                                </div>
                                                            </v-list-item-title>
                                                        </v-list-item>

                                                        <template :key="categoryType"
                                                                  v-for="(categories, categoryType) in allPrimaryCategories">
                                                            <v-list-item density="compact" v-show="categories && categories.length">
                                                                <v-list-item-title>
                                                                    <span class="text-sm">{{ getTransactionTypeName(categoryTypeToTransactionType(parseInt(categoryType)), 'Type') }}</span>
                                                                </v-list-item-title>
                                                            </v-list-item>

                                                            <v-list-group :key="category.id" v-for="category in categories">
                                                                <template #activator="{ props }" v-if="!category.hidden || queryAllFilterCategoryIds[category.id] || allCategories[query.categoryIds]?.parentId === category.id || hasSubCategoryInQuery(category)">
                                                                    <v-divider />
                                                                    <v-list-item class="text-sm" density="compact"
                                                                                 :class="getCategoryListItemCheckedClass(category, queryAllFilterCategoryIds)"
                                                                                 v-bind="props">
                                                                        <v-list-item-title>
                                                                            <div class="d-flex align-center">
                                                                                <ItemIcon icon-type="category" size="24px" :icon-id="category.icon" :color="category.color"></ItemIcon>
                                                                                <span class="text-sm ms-3">{{ category.name }}</span>
                                                                            </div>
                                                                        </v-list-item-title>
                                                                    </v-list-item>
                                                                </template>

                                                                <v-divider />
                                                                <v-list-item class="text-sm" density="compact"
                                                                             :class="{ 'item-in-multiple-selection': queryAllFilterCategoryIdsCount > 1 && queryAllFilterCategoryIds[category.id] }"
                                                                             :value="category.id"
                                                                             :append-icon="(query.categoryIds === category.id ? mdiCheck : undefined)">
                                                                    <v-list-item-title class="cursor-pointer"
                                                                                       @click="changeCategoryFilter(category.id)">
                                                                        <div class="d-flex align-center">
                                                                            <v-icon :icon="mdiViewGridOutline" />
                                                                            <span class="text-sm ms-3">{{ tt('All') }}</span>
                                                                        </div>
                                                                    </v-list-item-title>
                                                                </v-list-item>

                                                                <template :key="subCategory.id"
                                                                          v-for="subCategory in category.subCategories">
                                                                    <v-divider v-if="!subCategory.hidden || queryAllFilterCategoryIds[subCategory.id]" />
                                                                    <v-list-item class="text-sm" density="compact"
                                                                                 :value="subCategory.id"
                                                                                 :class="{ 'list-item-selected': query.categoryIds === subCategory.id, 'item-in-multiple-selection': queryAllFilterCategoryIdsCount > 1 && queryAllFilterCategoryIds[subCategory.id] }"
                                                                                 :append-icon="(query.categoryIds === subCategory.id ? mdiCheck : undefined)"
                                                                                 v-if="!subCategory.hidden || queryAllFilterCategoryIds[subCategory.id]">
                                                                        <v-list-item-title class="cursor-pointer"
                                                                                           @click="changeCategoryFilter(subCategory.id)">
                                                                            <div class="d-flex align-center">
                                                                                <ItemIcon icon-type="category" size="24px" :icon-id="subCategory.icon" :color="subCategory.color"></ItemIcon>
                                                                                <span class="text-sm ms-3">{{ subCategory.name }}</span>
                                                                            </div>
                                                                        </v-list-item-title>
                                                                    </v-list-item>
                                                                </template>
                                                            </v-list-group>
                                                        </template>
                                                    </v-list>
                                                </v-menu>
                                            </th>
                                            <th class="transaction-table-column-time text-no-wrap">
                                                <v-menu ref="timeFilterMenu" class="transaction-time-menu"
                                                        eager location="bottom" max-height="500"
                                                        @update:model-value="scrollTimeMenuToSelectedItem">
                                                    <template #activator="{ props }">
                                                        <div class="d-flex align-center cursor-pointer"
                                                             :class="{ 'readonly': loading, 'text-primary': query.dateType !== DateRange.ThisMonth.type }" v-bind="props">
                                                            <span>{{ tt('Time') }}</span>
                                                            <v-icon :icon="mdiMenuDown" />
                                                        </div>
                                                    </template>
                                                    <v-list :selected="[query.dateType]">
                                                        <v-list-item class="text-sm" density="compact"
                                                                     :key="dateRange.type" :value="dateRange.type"
                                                                     :class="{ 'list-item-selected': query.dateType === dateRange.type }"
                                                                     :append-icon="(query.dateType === dateRange.type ? mdiCheck : undefined)"
                                                                     v-for="dateRange in allDateRanges">
                                                            <v-list-item-title class="cursor-pointer"
                                                                               @click="changeDateFilter(dateRange.type)">
                                                                <div class="d-flex align-center">
                                                                    <span class="text-sm ms-3">{{ dateRange.displayName }}</span>
                                                                </div>
                                                                <div class="transaction-list-custom-datetime-range ms-3 smaller" v-if="dateRange.isUserCustomRange && query.dateType === dateRange.type && query.minTime && query.maxTime">
                                                                    <span>{{ queryMinTime }}</span>
                                                                    <span>&nbsp;-&nbsp;</span>
                                                                    <br/>
                                                                    <span>{{ queryMaxTime }}</span>
                                                                </div>
                                                            </v-list-item-title>
                                                        </v-list-item>
                                                    </v-list>
                                                </v-menu>
                                            </th>
                                            <th class="transaction-table-column-account text-no-wrap d-none">
                                                <v-menu ref="accountFilterMenu" class="transaction-account-menu"
                                                        eager location="bottom" max-height="500"
                                                        @update:model-value="scrollAccountMenuToSelectedItem">
                                                    <template #activator="{ props }">
                                                        <div class="d-flex align-center cursor-pointer"
                                                             :class="{ 'readonly': loading, 'text-primary': query.accountIds }" v-bind="props">
                                                            <span>{{ queryAccountName }}</span>
                                                            <v-icon :icon="mdiMenuDown" />
                                                        </div>
                                                    </template>
                                                    <v-list :selected="[queryAllSelectedFilterAccountIds]">
                                                        <v-list-item key="" value="" class="text-sm" density="compact"
                                                                     :class="{ 'list-item-selected': !query.accountIds }"
                                                                     :append-icon="(!query.accountIds ? mdiCheck : undefined)">
                                                            <v-list-item-title class="cursor-pointer"
                                                                               @click="changeAccountFilter('')">
                                                                <div class="d-flex align-center">
                                                                    <v-icon :icon="mdiViewGridOutline" />
                                                                    <span class="text-sm ms-3">{{ tt('All') }}</span>
                                                                </div>
                                                            </v-list-item-title>
                                                        </v-list-item>
                                                        <v-list-item key="multiple" value="multiple" class="text-sm" density="compact"
                                                                     :class="{ 'list-item-selected': query.accountIds && queryAllFilterAccountIdsCount > 1 }"
                                                                     :append-icon="(query.accountIds && queryAllFilterAccountIdsCount > 1 ? mdiCheck : undefined)"
                                                                     v-if="allAvailableAccountsCount > 0">
                                                            <v-list-item-title class="cursor-pointer"
                                                                               @click="showFilterAccountDialog = true">
                                                                <div class="d-flex align-center">
                                                                    <v-icon :icon="mdiVectorArrangeBelow" />
                                                                    <span class="text-sm ms-3">{{ tt('Multiple Accounts') }}</span>
                                                                </div>
                                                            </v-list-item-title>
                                                        </v-list-item>
                                                        <template :key="account.id"
                                                                  v-for="account in allAccounts">
                                                            <v-divider v-if="(!account.hidden && (!allAccountsMap[account.parentId] || !allAccountsMap[account.parentId]!.hidden)) || queryAllFilterAccountIds[account.id]" />
                                                            <v-list-item class="text-sm" density="compact"
                                                                         :value="account.id"
                                                                         :class="{ 'list-item-selected': query.accountIds === account.id, 'item-in-multiple-selection': queryAllFilterAccountIdsCount > 1 && queryAllFilterAccountIds[account.id] }"
                                                                         :append-icon="(query.accountIds === account.id ? mdiCheck : undefined)"
                                                                         v-if="(!account.hidden && (!allAccountsMap[account.parentId] || !allAccountsMap[account.parentId]!.hidden)) || queryAllFilterAccountIds[account.id]">
                                                                <v-list-item-title class="cursor-pointer"
                                                                                   @click="changeAccountFilter(account.id)">
                                                                    <div class="d-flex align-center">
                                                                        <ItemIcon icon-type="account" size="24px" :icon-id="account.icon" :color="account.color"></ItemIcon>
                                                                        <span class="text-sm ms-3">{{ account.name }}</span>
                                                                    </div>
                                                                </v-list-item-title>
                                                            </v-list-item>
                                                        </template>
                                                    </v-list>
                                                </v-menu>
                                            </th>
                                            <th class="transaction-table-column-tags text-no-wrap d-none" v-if="showTagInTransactionListPage">
                                                <v-menu ref="tagFilterMenu" class="transaction-tag-menu"
                                                        eager location="bottom" max-height="500"
                                                        @update:model-value="scrollTagMenuToSelectedItem">
                                                    <template #activator="{ props }">
                                                        <div class="d-flex align-center cursor-pointer"
                                                             :class="{ 'readonly': loading, 'text-primary': query.tagFilter }" v-bind="props">
                                                            <span>{{ queryTagName }}</span>
                                                            <v-icon :icon="mdiMenuDown" />
                                                        </div>
                                                    </template>
                                                    <v-list :selected="[queryAllSelectedFilterTagIds]">
                                                        <v-list-item key="" value="" class="text-sm" density="compact"
                                                                     :class="{ 'list-item-selected': !query.tagFilter }"
                                                                     :append-icon="(!query.tagFilter ? mdiCheck : undefined)">
                                                            <v-list-item-title class="cursor-pointer"
                                                                               @click="changeTagFilter('')">
                                                                <div class="d-flex align-center">
                                                                    <v-icon :icon="mdiViewGridOutline" />
                                                                    <span class="text-sm ms-3">{{ tt('All') }}</span>
                                                                </div>
                                                            </v-list-item-title>
                                                        </v-list-item>
                                                        <v-list-item class="text-sm" density="compact"
                                                                     :key="TransactionTagFilter.TransactionNoTagFilterValue"
                                                                     :value="TransactionTagFilter.TransactionNoTagFilterValue"
                                                                     :class="{ 'list-item-selected': query.tagFilter === TransactionTagFilter.TransactionNoTagFilterValue }"
                                                                     :append-icon="(query.tagFilter === TransactionTagFilter.TransactionNoTagFilterValue ? mdiCheck : undefined)">
                                                            <v-list-item-title class="cursor-pointer"
                                                                               @click="changeTagFilter(TransactionTagFilter.TransactionNoTagFilterValue)">
                                                                <div class="d-flex align-center">
                                                                    <v-icon :icon="mdiBorderNoneVariant" />
                                                                    <span class="text-sm ms-3">{{ tt('Without Tags') }}</span>
                                                                </div>
                                                            </v-list-item-title>
                                                        </v-list-item>
                                                        <v-list-item key="multiple" value="multiple" class="text-sm" density="compact"
                                                                     :class="{ 'list-item-selected': query.tagFilter && queryAllFilterTagIdsCount > 1 }"
                                                                     :append-icon="(query.tagFilter && queryAllFilterTagIdsCount > 1 ? mdiCheck : undefined)"
                                                                     v-if="allAvailableTagsCount > 0">
                                                            <v-list-item-title class="cursor-pointer"
                                                                               @click="showFilterTagDialog = true">
                                                                <div class="d-flex align-center">
                                                                    <v-icon :icon="mdiVectorArrangeBelow" />
                                                                    <span class="text-sm ms-3">{{ tt('Multiple Tags') }}</span>
                                                                </div>
                                                            </v-list-item-title>
                                                        </v-list-item>

                                                        <v-divider v-if="query.tagFilter && query.tagFilter !== TransactionTagFilter.TransactionNoTagFilterValue" />

                                                        <template :key="transactionTag.id"
                                                                  v-for="transactionTag in allTransactionTags">
                                                            <v-divider v-if="!transactionTag.hidden || isDefined(queryAllFilterTagIds[transactionTag.id])" />
                                                            <v-list-item class="text-sm" density="compact"
                                                                         :value="transactionTag.id"
                                                                         :class="{ 'list-item-selected': queryAllFilterTagIdsCount === 1 && isDefined(queryAllFilterTagIds[transactionTag.id]), 'item-in-multiple-selection': queryAllFilterTagIdsCount > 1 && isDefined(queryAllFilterTagIds[transactionTag.id]) }"
                                                                         :append-icon="(queryAllFilterTagIds[transactionTag.id] === true ? mdiCheck : (queryAllFilterTagIds[transactionTag.id] === false ? mdiClose : undefined))"
                                                                         v-if="!transactionTag.hidden || isDefined(queryAllFilterTagIds[transactionTag.id])">
                                                                <v-list-item-title class="cursor-pointer"
                                                                                   @click="changeTagFilter(TransactionTagFilter.of(transactionTag.id).toTextualTagFilter())">
                                                                    <div class="d-flex align-center">
                                                                        <v-icon size="24" :icon="mdiPound"/>
                                                                        <span class="text-sm ms-3">{{ transactionTag.name }}</span>
                                                                    </div>
                                                                </v-list-item-title>
                                                            </v-list-item>
                                                        </template>
                                                    </v-list>
                                                </v-menu>
                                            </th>
                                        </tr>
                                        </thead>

                                        <tbody v-if="pageType !== analysisPageType && loading && (!transactions || !transactions.length || transactions.length < 1)">
                                        <tr :key="itemIdx" v-for="itemIdx in skeletonData">
                                            <td class="px-0" colspan="4">
                                                <v-skeleton-loader type="text" :loading="true"></v-skeleton-loader>
                                            </td>
                                        </tr>
                                        </tbody>

                                        <tbody v-if="pageType !== analysisPageType && !loading && (!transactions || !transactions.length || transactions.length < 1)">
                                        <tr>
                                            <td colspan="4">{{ tt('No transaction data') }}</td>
                                        </tr>
                                        </tbody>

                                        <template v-if="pageType === TransactionListPageType.Calendar.type">
                                        <tbody :key="transaction.id"
                                               :class="{ 'disabled': loading, 'has-bottom-border': idx < transactions.length - 1 }"
                                               v-for="(transaction, idx) in transactions">
                                            <tr class="transaction-list-row-date no-hover text-sm"
                                                v-if="pageType === TransactionListPageType.List.type && (idx === 0 || (idx > 0 && (transaction.gregorianCalendarYearDashMonthDashDay !== transactions[idx - 1]!.gregorianCalendarYearDashMonthDashDay)))">
                                                <td colspan="4" class="font-weight-bold">
                                                    <div class="d-flex align-center">
                                                        <span>{{ getDisplayLongDate(transaction) }}</span>
                                                        <v-chip class="ms-1" color="default" size="x-small"
                                                                v-if="transaction.displayDayOfWeek">
                                                            {{ getWeekdayLongName(transaction.displayDayOfWeek) }}
                                                        </v-chip>
                                                    </div>
                                                </td>
                                            </tr>
                                            <tr class="transaction-table-row-data text-sm cursor-pointer"
                                                @click="show(transaction)">
                                                <td class="transaction-table-column-description text-truncate">
                                                    {{ transaction.comment }}
                                                </td>
                                                <td class="transaction-table-column-amount" :class="{ 'text-expense': transaction.type === TransactionType.Expense, 'text-income': transaction.type === TransactionType.Income }">
                                                    <div v-if="transaction.isCashFlow || transaction.sourceAccount">
                                                        <span>{{ getDisplayAmount(transaction) }}</span>
                                                    </div>
                                                </td>
                                                <td class="transaction-table-column-category">
                                                    <div class="d-flex align-center">
                                                        <ItemIcon size="24px" icon-type="category"
                                                                  :icon-id="transaction.category.icon"
                                                                  :color="transaction.category.color"
                                                                  v-if="transaction.category && transaction.category.color"></ItemIcon>
                                                        <v-icon size="24" :icon="mdiPencilBoxOutline" v-else-if="!transaction.category || !transaction.category.color" />
                                                        <span class="ms-2" v-if="transaction.type === TransactionType.ModifyBalance">
                                                            {{ tt('Modify Balance') }}
                                                        </span>
                                                        <span class="ms-2" v-else-if="transaction.type !== TransactionType.ModifyBalance && transaction.category">
                                                            {{ transaction.category.name }}
                                                        </span>
                                                        <span class="ms-2" v-else-if="transaction.type !== TransactionType.ModifyBalance && !transaction.category">
                                                            {{ getTransactionTypeName(transaction.type, 'Transaction') }}
                                                        </span>
                                                    </div>
                                                </td>
                                                <td class="transaction-table-column-time">
                                                    <div class="d-flex flex-column">
                                                        <span>{{ getDisplayTime(transaction) }}</span>
                                                        <span class="text-caption" v-if="!isSameAsDefaultTimezoneOffsetMinutes(transaction)">{{ getDisplayTimezone(transaction) }}</span>
                                                        <v-tooltip activator="parent" v-if="!isSameAsDefaultTimezoneOffsetMinutes(transaction)">{{ getDisplayTimeInDefaultTimezone(transaction) }}</v-tooltip>
                                                    </div>
                                                </td>
                                                <td class="transaction-table-column-account d-none">
                                                    <div class="d-flex align-center">
                                                        <span v-if="transaction.sourceAccount">{{ transaction.sourceAccount.name }}</span>
                                                        <v-icon class="icon-with-direction mx-1" size="13" :icon="mdiArrowRight" v-if="transaction.sourceAccount && transaction.type === TransactionType.Transfer && transaction.destinationAccount && transaction.sourceAccount.id !== transaction.destinationAccount.id"></v-icon>
                                                        <span v-if="transaction.sourceAccount && transaction.type === TransactionType.Transfer && transaction.destinationAccount && transaction.sourceAccount.id !== transaction.destinationAccount.id">{{ transaction.destinationAccount.name }}</span>
                                                    </div>
                                                </td>
                                                <td class="transaction-table-column-tags d-none" v-if="showTagInTransactionListPage">
                                                    <v-chip class="transaction-tag" size="small" :prepend-icon="mdiPound"
                                                            :text="allTransactionTags[tagId]?.name"
                                                            :key="tagId"
                                                            v-for="tagId in transaction.tagIds"/>
                                                    <v-chip class="transaction-tag" size="small"
                                                            :text="tt('None')"
                                                            v-if="!transaction.tagIds || !transaction.tagIds.length"/>
                                                </td>
                                            </tr>
                                        </tbody>
                                        </template>
                                    </v-table>

                                    <transaction-analysis v-if="pageType === analysisPageType"
                                                          :transactions="transactions" :loading="loading"
                                                          :default-currency="defaultCurrency"
                                                          :show-amounts="showTotalAmountInTransactionListPage"
                                                          @select="show" />

                                    <div v-if="pageType === TransactionListPageType.List.type && transactions.length"
                                         :class="{ 'transaction-category-loading': loading }" :aria-busy="loading">
                                        <section v-for="day in transactionCategoryDays" :key="day.date" :aria-label="getDisplayLongDate(day.firstTransaction)">
                                            <div class="transaction-category-date">
                                                {{ getDisplayLongDate(day.firstTransaction) }}
                                                <v-chip class="ms-1" color="default" size="x-small" v-if="day.firstTransaction.displayDayOfWeek">
                                                    {{ getWeekdayLongName(day.firstTransaction.displayDayOfWeek) }}
                                                </v-chip>
                                                <span class="transaction-daily-total">
                                                    {{ queryAllFilterAccountIdsCount ? tt('Total Inflows') : tt('Total Income') }}
                                                    <span class="text-income">{{ getDisplayMonthTotalAmount(day.totalAmount.income, defaultCurrency, '', day.totalAmount.incompleteIncome) }}</span>
                                                </span>
                                                <span class="transaction-daily-total">
                                                    {{ queryAllFilterAccountIdsCount ? tt('Total Outflows') : tt('Total Expense') }}
                                                    <span class="text-expense">{{ getDisplayMonthTotalAmount(day.totalAmount.expense, defaultCurrency, '', day.totalAmount.incompleteExpense) }}</span>
                                                </span>
                                            </div>
                                            <div class="transaction-category-scroll" tabindex="0" :aria-label="getDisplayLongDate(day.firstTransaction)">
                                                <div class="transaction-category-grid" :style="{ gridTemplateColumns: `repeat(${day.categories.length}, 280px)` }">
                                                    <div class="transaction-category-cell" v-for="column in day.categories" :key="column.key">
                                                        <div class="transaction-category-heading">
                                                            <ItemIcon size="20px" icon-type="category" :icon-id="column.transaction.category.icon"
                                                                      :color="column.transaction.category.color" v-if="column.transaction.category" />
                                                            <v-icon :icon="mdiPencilBoxOutline" size="20" v-else />
                                                            <span class="text-truncate">{{ column.transaction.category?.name || getTransactionTypeName(column.transaction.type, 'Transaction') }}</span>
                                                        </div>
                                                        <ul class="transaction-category-list">
                                                            <li v-for="transaction in column.items" :key="transaction.id">
                                                                <button class="transaction-category-item" type="button"
                                                                        :disabled="loading" @click="show(transaction)">
                                                                    <span class="transaction-category-description text-truncate" :title="transaction.comment">{{ transaction.comment || column.transaction.category?.name || getTransactionTypeName(transaction.type, 'Transaction') }}</span>
                                                                    <span class="transaction-category-details">
                                                                        <span :class="{ 'text-expense': transaction.type === TransactionType.Expense, 'text-income': transaction.type === TransactionType.Income }">{{ getDisplayAmount(transaction) }}</span>
                                                                        <span class="transaction-category-time">
                                                                            {{ getDisplayTime(transaction) }}
                                                                            <span v-if="!isSameAsDefaultTimezoneOffsetMinutes(transaction)">{{ getDisplayTimezone(transaction) }}</span>
                                                                            <v-tooltip activator="parent" v-if="!isSameAsDefaultTimezoneOffsetMinutes(transaction)">{{ getDisplayTimeInDefaultTimezone(transaction) }}</v-tooltip>
                                                                        </span>
                                                                    </span>
                                                                </button>
                                                            </li>
                                                        </ul>
                                                    </div>
                                                </div>
                                            </div>
                                        </section>
                                    </div>

                                    <v-infinite-scroll v-if="pageType === TransactionListPageType.List.type && !queryMonthlyData && !loading && !refreshingList"
                                                       :key="loadGeneration" :empty-text="''" @load="loadMoreTransactions" />
                                </v-card>
                            </v-window-item>
                        </v-window>
                    </v-main>
                </v-layout>
            </v-card>
        </v-col>
    </v-row>

    <date-range-selection-dialog :title="tt('Custom Date Range')"
                                 :min-time="customMinDatetime"
                                 :max-time="customMaxDatetime"
                                 v-model:show="showCustomDateRangeDialog"
                                 @dateRange:change="changeCustomDateFilter"
                                 @error="onShowDateRangeError" />

    <month-selection-dialog :title="tt('Select Month')"
                            :model-value="queryMonth"
                            v-model:show="showCustomMonthDialog"
                            @update:modelValue="changeCustomMonthDateFilter"
                            @error="onShowDateRangeError" />

    <edit-dialog ref="editDialog" :type="TransactionEditPageType.Transaction" />
    <a-i-image-recognition-dialog ref="aiImageRecognitionDialog" />
    <import-dialog ref="importDialog" :persistent="true" />

    <v-dialog width="800" v-model="showFilterAccountDialog">
        <account-filter-settings-card type="transactionListCurrent" :dialog-mode="true"
                                      @settings:change="changeMultipleAccountsFilter" />
    </v-dialog>

    <v-dialog width="800" v-model="showFilterCategoryDialog">
        <category-filter-settings-card type="transactionListCurrent" :dialog-mode="true" :category-types="allowCategoryTypes"
                                       @settings:change="changeMultipleCategoriesFilter" />
    </v-dialog>

    <v-dialog width="800" v-model="showFilterTagDialog">
        <transaction-tag-filter-settings-card type="transactionListCurrent" :dialog-mode="true"
                                       @settings:change="changeMultipleTagsFilter" />
    </v-dialog>

    <confirm-dialog ref="confirmDialog"/>
    <snack-bar ref="snackbar" />
</template>

<script setup lang="ts">
import { VMenu } from 'vuetify/components/VMenu';
import { VInfiniteScroll } from 'vuetify/components/VInfiniteScroll';
import ConfirmDialog from '@/components/desktop/ConfirmDialog.vue';
import SnackBar from '@/components/desktop/SnackBar.vue';
import EditDialog from './list/dialogs/EditDialog.vue';
import AIImageRecognitionDialog from './list/dialogs/AIImageRecognitionDialog.vue';
import ImportDialog from './import/ImportDialog.vue';
import AccountFilterSettingsCard from '@/views/desktop/common/cards/AccountFilterSettingsCard.vue';
import CategoryFilterSettingsCard from '@/views/desktop/common/cards/CategoryFilterSettingsCard.vue';
import TransactionTagFilterSettingsCard from '@/views/desktop/common/cards/TransactionTagFilterSettingsCard.vue';
import { TransactionEditPageType } from '@/views/base/transactions/TransactionEditPageBase.ts';

import TransactionAnalysis from '@/components/desktop/TransactionAnalysis.vue';

import { useCoalescedRefresh } from '@/components/base/CoalescedRefresh.ts';

import { ref, computed, useTemplateRef, watch, nextTick } from 'vue';
import { useRouter, onBeforeRouteUpdate } from 'vue-router';
import { useTheme } from 'vuetify';

import { useI18n } from '@/locales/helpers.ts';
import { TransactionListPageType, useTransactionListPageBase } from '@/views/base/transactions/TransactionListPageBase.ts';

import { useSettingsStore } from '@/stores/setting.ts';
import { useUserStore } from '@/stores/user.ts';
import { useAccountsStore } from '@/stores/account.ts';
import { useTransactionCategoriesStore } from '@/stores/transactionCategory.ts';
import { useTransactionTagsStore } from '@/stores/transactionTag.ts';
import { useTransactionsStore } from '@/stores/transaction.ts';
import { useTransactionTemplatesStore } from '@/stores/transactionTemplate.ts';
import { useDesktopPageStore } from '@/stores/desktopPage.ts';

import {
    keys
} from '@/core/base.ts';
import {
    type Year0BasedMonth,
    type LocalizedRecentMonthDateRange,
    type TimeRangeAndDateType,
    DateRangeScene,
    DateRange
} from '@/core/datetime.ts';
import { AmountFilterType } from '@/core/numeral.ts';
import { ThemeType } from '@/core/theme.ts';
import { TransactionType } from '@/core/transaction.ts';
import { TemplateType }  from '@/core/template.ts';
import type { TransactionCategory } from '@/models/transaction_category.ts';
import { type Transaction, TransactionTagFilter } from '@/models/transaction.ts';
import type { TransactionTemplate } from '@/models/transaction_template.ts';

import {
    isDefined,
    isObject,
    isString,
    isNumber,
    objectFieldWithValueToArrayItem
} from '@/lib/common.ts';
import {
    getCurrentUnixTime,
    parseDateTimeFromUnixTime,
    getDayFirstUnixTimeBySpecifiedUnixTime,
    getYearMonthFirstUnixTime,
    getYearMonthLastUnixTime,
    getShiftedDateRangeAndDateType,
    getShiftedDateRangeAndDateTypeForBillingCycle,
    getDateTypeByDateRange,
    getDateTypeByBillingCycleDateRange,
    getDateRangeByDateType,
    getDateRangeByBillingCycleDateType,
    getRecentDateRangeIndex,
    getFullMonthDateRange,
    getValidMonthDayOrCurrentDayShortDate
} from '@/lib/datetime.ts';
import {
    categoryTypeToTransactionType,
    transactionTypeToCategoryType
} from '@/lib/category.ts';
import { isDataExportingEnabled, isDataImportingEnabled, isTransactionFromAIImageRecognitionEnabled } from '@/lib/server_settings.ts';
import { isSpeechRecognitionSupported } from '@/lib/voice.ts';
import { recognizeTransactionByVoice } from '@/lib/transaction_voice.ts';
import { scrollToSelectedItem, startDownloadFile } from '@/lib/ui/common.ts';
import logger from '@/lib/logger.ts';

import {
    mdiMagnify,
    mdiCheck,
    mdiClose,
    mdiViewGridOutline,
    mdiBorderNoneVariant,
    mdiVectorArrangeBelow,
    mdiRefresh,
    mdiTuneVariant,
    mdiDotsHorizontal,
    mdiPlus,
    mdiImport,
    mdiChevronLeft,
    mdiChevronRight,
    mdiChevronDown,
    mdiMenuDown,
    mdiPencilBoxOutline,
    mdiArrowRight,
    mdiPound,
    mdiMagicStaff,
    mdiMicrophone,
    mdiTextBoxOutline
} from '@mdi/js';

interface TransactionListProps {
    initPageType?: string;
    initDateType?: string,
    initMaxTime?: string,
    initMinTime?: string,
    initType?: string,
    initCategoryIds?: string,
    initAccountIds?: string,
    initTagFilter?: string,
    initAmountFilter?: string,
    initKeyword?: string
}

const props = defineProps<TransactionListProps>();
const analysisPageType = 2;
const desktopPageTypes = [...TransactionListPageType.values(), { type: analysisPageType, name: 'Transaction Analysis' }];

type ConfirmDialogType = InstanceType<typeof ConfirmDialog>;
type SnackBarType = InstanceType<typeof SnackBar>;
type EditDialogType = InstanceType<typeof EditDialog>;
type AIImageRecognitionDialogType = InstanceType<typeof AIImageRecognitionDialog>;
type ImportDialogType = InstanceType<typeof ImportDialog>;

interface TransactionListDisplayTotalAmount {
    income: string;
    expense: string;
}

const router = useRouter();
const theme = useTheme();

const {
    tt,
    getAllRecentMonthDateRanges,
    getWeekdayLongName,
    getCurrentLanguageInfo
} = useI18n();

const {
    pageType,
    loading,
    customMinDatetime,
    customMaxDatetime,
    currentCalendarDate,
    firstDayOfWeek,
    fiscalYearStart,
    defaultCurrency,
    showTotalAmountInTransactionListPage,
    showTagInTransactionListPage,
    allDateRanges,
    allAccounts,
    allAccountsMap,
    allAvailableAccountsCount,
    allCategories,
    allPrimaryCategories,
    allAvailableCategoriesCount,
    allTransactionTags,
    allAvailableTagsCount,
    query,
    queryMinTime,
    queryMaxTime,
    queryMonthlyData,
    queryMonth,
    queryAllFilterCategoryIds,
    queryAllFilterAccountIds,
    queryAllFilterTagIds,
    queryAllFilterCategoryIdsCount,
    queryAllFilterAccountIdsCount,
    queryAllFilterTagIdsCount,
    queryAccountName,
    queryCategoryName,
    queryTagName,
    queryAmount,
    transactionCalendarMinDate,
    transactionCalendarMaxDate,
    currentMonthTransactionData,
    hasSubCategoryInQuery,
    isSameAsDefaultTimezoneOffsetMinutes,
    canAddTransaction,
    getDisplayTime,
    getDisplayLongDate,
    getDisplayTimezone,
    getDisplayTimeInDefaultTimezone,
    getDisplayAmount,
    getDisplayMonthTotalAmount,
    getTransactionTypeName,
} = useTransactionListPageBase();

const settingsStore = useSettingsStore();
const userStore = useUserStore();
const accountsStore = useAccountsStore();
const transactionCategoriesStore = useTransactionCategoriesStore();
const transactionTagsStore = useTransactionTagsStore();
const transactionsStore = useTransactionsStore();
const transactionTemplatesStore = useTransactionTemplatesStore();
const desktopPageStore = useDesktopPageStore();

const timeFilterMenu = useTemplateRef<VMenu>('timeFilterMenu');
const categoryFilterMenu = useTemplateRef<VMenu>('categoryFilterMenu');
const amountFilterMenu = useTemplateRef<VMenu>('amountFilterMenu');
const accountFilterMenu = useTemplateRef<VMenu>('accountFilterMenu');
const tagFilterMenu = useTemplateRef<VMenu>('tagFilterMenu');

const confirmDialog = useTemplateRef<ConfirmDialogType>('confirmDialog');
const snackbar = useTemplateRef<SnackBarType>('snackbar');
const editDialog = useTemplateRef<EditDialogType>('editDialog');
const aiImageRecognitionDialog = useTemplateRef<AIImageRecognitionDialogType>('aiImageRecognitionDialog');
const importDialog = useTemplateRef<ImportDialogType>('importDialog');

const activeTab = ref<string>('transactionPage');
const currentPage = ref<number>(1);
const loadGeneration = ref(0);
const loadingMore = ref(false);
const refreshingList = ref(false);
const totalCount = ref<number>(1);
const searchKeyword = ref<string>('');
const currentAmountFilterType = ref<string>('');
const currentAmountFilterValue1 = ref<number>(0);
const currentAmountFilterValue2 = ref<number>(0);
const currentPageTransactions = ref<Transaction[]>([]);
const categoryMenuState = ref<boolean>(false);
const amountMenuState = ref<boolean>(false);
const exportingData = ref<boolean>(false);
const recognizingByVoice = ref<boolean>(false);
const showCustomDateRangeDialog = ref<boolean>(false);
const showCustomMonthDialog = ref<boolean>(false);
const showFilterAccountDialog = ref<boolean>(false);
const showFilterCategoryDialog = ref<boolean>(false);
const showFilterTagDialog = ref<boolean>(false);

const isDarkMode = computed<boolean>(() => theme.global.name.value === ThemeType.Dark);

const recentMonthDateRanges = computed<LocalizedRecentMonthDateRange[]>(() => getAllRecentMonthDateRanges(pageType.value !== TransactionListPageType.Calendar.type, true));

const allTransactionTemplates = computed<TransactionTemplate[]>(() => {
    const allTemplates = transactionTemplatesStore.allVisibleTemplates;
    return allTemplates[TemplateType.Normal.type] || [];
});

const allowCategoryTypes = computed<string>(() => {
    if (TransactionType.Income <= query.value.type && query.value.type <= TransactionType.Transfer) {
        return transactionTypeToCategoryType(query.value.type)?.toString() ?? '';
    }

    return '';
});

const transactions = computed<Transaction[]>(() => {
    if (pageType.value === TransactionListPageType.List.type || pageType.value === analysisPageType) {
        if (queryMonthlyData.value) {
            const transactionData = currentMonthTransactionData.value;

            if (!transactionData || !transactionData.items) {
                return [];
            }

            return transactionData.items.filter(transaction => transaction.type !== TransactionType.ModifyBalance);
        } else {
            return currentPageTransactions.value.filter(transaction => transaction.type !== TransactionType.ModifyBalance);
        }
    } else if (pageType.value === TransactionListPageType.Calendar.type) {
        if (queryMonthlyData.value) {
            const transactionData = currentMonthTransactionData.value;

            if (!transactionData || !transactionData.items) {
                return [];
            }

            const transactions :Transaction[] = [];

            for (const transaction of transactionData.items) {
                if (transaction.gregorianCalendarYearDashMonthDashDay === currentCalendarDate.value) {
                    transactions.push(transaction);
                }
            }

            return transactions;
        } else {
            return [];
        }
    } else {
        return [];
    }
});

function getTransactionCategoryColumnKey(transaction: Transaction): string {
    return `${transaction.type}:${transaction.categoryId || '0'}`;
}

const transactionCategoryDays = computed(() => {
    const days = new Map<string, { date: string; firstTransaction: Transaction; columns: Map<string, Transaction[]> }>();

    for (const transaction of transactions.value) {
        const date = transaction.gregorianCalendarYearDashMonthDashDay || getDisplayLongDate(transaction);
        let day = days.get(date);
        if (!day) {
            day = { date, firstTransaction: transaction, columns: new Map() };
            days.set(date, day);
        }
        const key = getTransactionCategoryColumnKey(transaction);
        const items = day.columns.get(key) || [];
        items.push(transaction);
        day.columns.set(key, items);
    }

    return [...days.values()].map((day, index) => ({
        ...day,
        categories: [...day.columns.entries()].map(([key, items]) => ({ key, items, transaction: items[0]! })).sort((a, b) =>
            a.transaction.type - b.transaction.type ||
            (a.transaction.category?.displayOrder ?? 0) - (b.transaction.category?.displayOrder ?? 0) ||
            a.key.localeCompare(b.key)),
        totalAmount: transactionsStore.calculateTransactionTotalAmount(
            [...day.columns.values()].flat(),
            defaultCurrency.value,
            query.value.accountIds,
            !queryMonthlyData.value && currentPageTransactions.value.length < totalCount.value && index === days.size - 1
        )
    }));
});

const recentDateRangeIndex = computed<number>({
    get: () => getRecentDateRangeIndex(recentMonthDateRanges.value, query.value.dateType, query.value.minTime, query.value.maxTime, firstDayOfWeek.value, fiscalYearStart.value),
    set: (value) => {
        if (value < 0 || value >= recentMonthDateRanges.value.length) {
            value = 0;
        }

        changeDateFilter(recentMonthDateRanges.value[value] as LocalizedRecentMonthDateRange);
    }
});

const queryPageType = computed<number>({
    get: () => pageType.value,
    set: (value) => changePageType(value)
});

const queryType = computed<number>({
    get: () => query.value.type,
    set: (value) => changeTypeFilter(value)
});

const queryAllSelectedFilterCategoryIds = computed<string>(() => {
    if (queryAllFilterCategoryIdsCount.value === 0) {
        return '';
    } else if (queryAllFilterCategoryIdsCount.value === 1) {
        return query.value.categoryIds;
    } else { // queryAllFilterCategoryIdsCount.value > 1
        return 'multiple';
    }
});

const queryAllSelectedFilterAccountIds = computed<string>(() => {
    if (queryAllFilterAccountIdsCount.value === 0) {
        return '';
    } else if (queryAllFilterAccountIdsCount.value === 1) {
        return query.value.accountIds;
    } else { // queryAllFilterAccountIdsCount.value > 1
        return 'multiple';
    }
});

const queryAllSelectedFilterTagIds = computed<string>(() => {
    if (query.value.tagFilter === TransactionTagFilter.TransactionNoTagFilterValue) {
        return TransactionTagFilter.TransactionNoTagFilterValue;
    } else if (queryAllFilterTagIdsCount.value === 0) {
        return '';
    } else if (queryAllFilterTagIdsCount.value === 1) {
        for (const tagId of keys(queryAllFilterTagIds.value)) {
            return tagId;
        }

        return '';
    } else { // queryAllFilterTagIdsCount.value > 1
        return 'multiple';
    }
});

const countPerPage = computed<number>(() => settingsStore.appSettings.itemsCountInTransactionListPage);

const skeletonData = computed<number[]>(() => {
    const data: number[] = [];
    const totalCount = (pageType.value === TransactionListPageType.List.type ? countPerPage.value : 3);

    for (let i = 0; i < totalCount; i++) {
        data.push(i);
    }

    return data;
});

const currentMonthTotalAmount = computed<TransactionListDisplayTotalAmount | null>(() => {
    if (queryMonthlyData.value) {
        const transactionData = currentMonthTransactionData.value;

        if (!transactionData) {
            return null;
        }

        return {
            income: getDisplayMonthTotalAmount(transactionData.totalAmount.income, defaultCurrency.value, '', transactionData.totalAmount.incompleteIncome),
            expense: getDisplayMonthTotalAmount(transactionData.totalAmount.expense, defaultCurrency.value, '', transactionData.totalAmount.incompleteExpense)
        };
    } else {
        return null;
    }
});

function getCategoryListItemCheckedClass(category: TransactionCategory, queryCategoryIds: Record<string, boolean>): Record<string, boolean> {
    if (queryCategoryIds && queryCategoryIds[category.id]) {
        return {
            'list-item-selected': true,
            'has-children-item-selected': true
        };
    }

    if (category.subCategories) {
        for (const subCategory of category.subCategories) {
            if (queryCategoryIds && queryCategoryIds[subCategory.id]) {
                return {
                    'list-item-selected': true,
                    'has-children-item-selected': true
                };
            }
        }
    }

    return {};
}

function getAmountFilterParameterCount(filterType: string): number {
    const amountFilterType = AmountFilterType.valueOf(filterType);
    return amountFilterType ? amountFilterType.paramCount : 0;
}

function updateUrlWhenChanged(changed: boolean): void {
    if (changed) {
        loading.value = true;
        currentPageTransactions.value = [];
        transactionsStore.clearTransactions();
        router.push(`/transaction/list?${transactionsStore.getTransactionListPageParams(pageType.value)}`);
    }
}

function init(initProps: TransactionListProps): void {
    let dateRange: TimeRangeAndDateType | null = getDateRangeByDateType(initProps.initDateType ? parseInt(initProps.initDateType) : undefined, firstDayOfWeek.value, fiscalYearStart.value);

    if (!dateRange && initProps.initDateType && initProps.initMaxTime && initProps.initMinTime &&
        (DateRange.isBillingCycle(parseInt(initProps.initDateType)) || initProps.initDateType === DateRange.Custom.type.toString()) &&
        parseInt(initProps.initMaxTime) > 0 && parseInt(initProps.initMinTime) > 0) {
        dateRange = {
            dateType: parseInt(initProps.initDateType),
            maxTime: parseInt(initProps.initMaxTime),
            minTime: parseInt(initProps.initMinTime)
        };
    }

    transactionsStore.initTransactionListFilter({
        dateType: dateRange ? dateRange.dateType : undefined,
        maxTime: dateRange ? dateRange.maxTime : undefined,
        minTime: dateRange ? dateRange.minTime : undefined,
        type: initProps.initType && parseInt(initProps.initType) > 0 ? parseInt(initProps.initType) : undefined,
        categoryIds: initProps.initCategoryIds,
        accountIds: initProps.initAccountIds,
        tagFilter: initProps.initTagFilter,
        amountFilter: initProps.initAmountFilter || '',
        keyword: initProps.initKeyword || ''
    });

    if (initProps.initPageType) {
        const type = desktopPageTypes.find(item => item.type === parseInt(initProps.initPageType!));

        if (type) {
            pageType.value = type.type;
            currentCalendarDate.value = getValidMonthDayOrCurrentDayShortDate(query.value.minTime, currentCalendarDate.value);

            if (pageType.value === TransactionListPageType.Calendar.type) {
                const dateRange = getFullMonthDateRange(query.value.minTime, query.value.maxTime, firstDayOfWeek.value, fiscalYearStart.value);

                if (dateRange) {
                    const changed = transactionsStore.updateTransactionListFilter({
                        dateType: dateRange.dateType,
                        maxTime: dateRange.maxTime,
                        minTime: dateRange.minTime
                    });

                    if (changed) {
                        currentCalendarDate.value = getValidMonthDayOrCurrentDayShortDate(query.value.minTime, currentCalendarDate.value);
                        updateUrlWhenChanged(changed);
                        return;
                    }
                }
            }
        }
    }

    searchKeyword.value = initProps.initKeyword || '';
    currentAmountFilterType.value = '';

    currentPage.value = 1;
    reload(false, true);

    transactionTemplatesStore.loadAllTemplates({
        templateType: TemplateType.Normal.type,
        force: false
    });
}

function reload(force: boolean, init: boolean, background = false): Promise<void> {
    const pagesToLoad = pageType.value === analysisPageType ? Number.POSITIVE_INFINITY : (background ? currentPage.value : 1);
    refreshingList.value = background;
    if (!background) {
        loading.value = true;
        currentPage.value = 1;
    }
    loadingMore.value = false;
    const generation = ++loadGeneration.value;
    const page = 1;

    return Promise.all([
        accountsStore.loadAllAccounts({ force: false }),
        transactionCategoriesStore.loadAllCategories({ force: false }),
        transactionTagsStore.loadAllTags({ force: false })
    ]).then(async () => {
        if (generation !== loadGeneration.value) { return; }
        if (init) {
            if (desktopPageStore.showAddTransactionDialogInTransactionList) {
                desktopPageStore.resetShowAddTransactionDialogInTransactionList();
                add();
            }
        }

        if (queryMonthlyData.value) {
            const currentMonthMinDate = parseDateTimeFromUnixTime(query.value.minTime);
            const currentYear = currentMonthMinDate.getGregorianCalendarYear();
            const currentMonth = currentMonthMinDate.getGregorianCalendarMonth();

            return transactionsStore.loadMonthlyAllTransactions({
                year: currentYear,
                month: currentMonth,
                autoExpand: true,
                defaultCurrency: defaultCurrency.value
            });
        } else {
            const items: Transaction[] = [];
            let refreshedTotalCount = 0;
            for (let requestedPage = 1; requestedPage <= pagesToLoad; requestedPage++) {
                if (generation !== loadGeneration.value) { return; }
                const result = await transactionsStore.loadTransactions({
                    reload: true,
                    count: countPerPage.value,
                    page: requestedPage,
                    withCount: requestedPage === 1,
                    autoExpand: true,
                    defaultCurrency: defaultCurrency.value
                });
                items.push(...result.items);
                if (requestedPage === 1) refreshedTotalCount = result.totalCount || 0;
                if (!result.items.length || items.length >= refreshedTotalCount) break;
            }
            return { items, totalCount: refreshedTotalCount };
        }
    }).then(data => {
        if (generation !== loadGeneration.value) { return; }
        loading.value = false;
        currentPageTransactions.value = data && data.items && data.items.length ? data.items : [];

        if (page <= 1) {
            totalCount.value = data && data.totalCount ? data.totalCount : 1;
        }

        if (force) {
            snackbar.value?.showMessage('Data has been updated');
        }
    }).catch(error => {
        if (generation !== loadGeneration.value) { return; }
        loading.value = false;
        if (!background) {
            currentPageTransactions.value = [];
            totalCount.value = 1;
        }

        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    }).finally(() => {
        if (generation === loadGeneration.value) refreshingList.value = false;
    });
}

useCoalescedRefresh(
    () => desktopPageStore.transactionListRevision,
    () => loading.value || loadingMore.value || refreshingList.value,
    () => reload(false, false, true)
);

async function loadMoreTransactions({ done }: { done: (status: 'ok' | 'empty' | 'error') => void }): Promise<void> {
    if (loading.value || refreshingList.value || loadingMore.value || queryMonthlyData.value || currentPageTransactions.value.length >= totalCount.value) {
        done('empty');
        return;
    }

    const generation = loadGeneration.value;
    loadingMore.value = true;
    try {
        const data = await transactionsStore.loadTransactions({
            reload: true,
            count: countPerPage.value,
            page: currentPage.value + 1,
            withCount: false,
            autoExpand: true,
            defaultCurrency: defaultCurrency.value
        });
        if (generation !== loadGeneration.value) { done('empty'); return; }
        const existingIds = new Set(currentPageTransactions.value.map(transaction => transaction.id));
        currentPageTransactions.value.push(...data.items.filter(transaction => !existingIds.has(transaction.id)));
        currentPage.value++;
        done(!data.items.length || currentPageTransactions.value.length >= totalCount.value ? 'empty' : 'ok');
    } catch {
        if (generation !== loadGeneration.value) { done('empty'); return; }
        done('error');
        snackbar.value?.showError('Unable to retrieve transaction list');
    } finally {
        if (generation === loadGeneration.value) { loadingMore.value = false; }
    }
}

function changePageType(type: number): void {
    pageType.value = type;
    currentCalendarDate.value = getValidMonthDayOrCurrentDayShortDate(query.value.minTime, currentCalendarDate.value);

    if (pageType.value === TransactionListPageType.Calendar.type) {
        const dateRange = getFullMonthDateRange(query.value.minTime, query.value.maxTime, firstDayOfWeek.value, fiscalYearStart.value);

        if (dateRange) {
            transactionsStore.updateTransactionListFilter({
                dateType: dateRange.dateType,
                maxTime: dateRange.maxTime,
                minTime: dateRange.minTime
            });
            currentCalendarDate.value = getValidMonthDayOrCurrentDayShortDate(query.value.minTime, currentCalendarDate.value);
        }
    }

    updateUrlWhenChanged(true);
}

function changeDateFilter(dateRange: TimeRangeAndDateType | number | null): void {
    if (dateRange === DateRange.Custom.type || (isObject(dateRange) && dateRange.dateType === DateRange.Custom.type && !dateRange.minTime && !dateRange.maxTime)) { // Custom
        if (!query.value.minTime || !query.value.maxTime) {
            customMaxDatetime.value = getCurrentUnixTime();
            customMinDatetime.value = getDayFirstUnixTimeBySpecifiedUnixTime(customMaxDatetime.value);
        } else {
            customMaxDatetime.value = query.value.maxTime;
            customMinDatetime.value = query.value.minTime;
        }

        if (pageType.value === TransactionListPageType.Calendar.type) {
            showCustomMonthDialog.value = true;
        } else {
            showCustomDateRangeDialog.value = true;
        }

        return;
    }

    if (isNumber(dateRange)) {
        if (DateRange.isBillingCycle(dateRange)) {
            dateRange = getDateRangeByBillingCycleDateType(dateRange, firstDayOfWeek.value, fiscalYearStart.value, accountsStore.getAccountStatementDate(query.value.accountIds));
        } else {
            dateRange = getDateRangeByDateType(dateRange, firstDayOfWeek.value, fiscalYearStart.value);
        }
    }

    if (!dateRange) {
        return;
    }

    if (pageType.value === TransactionListPageType.Calendar.type) {
        currentCalendarDate.value = getValidMonthDayOrCurrentDayShortDate(dateRange.minTime, currentCalendarDate.value);
        const fullMonthDateRange = getFullMonthDateRange(dateRange.minTime, dateRange.maxTime, firstDayOfWeek.value, fiscalYearStart.value);

        if (fullMonthDateRange) {
            dateRange = fullMonthDateRange;
            currentCalendarDate.value = getValidMonthDayOrCurrentDayShortDate(dateRange.minTime, currentCalendarDate.value);
        }
    }

    if (query.value.dateType === dateRange.dateType && query.value.maxTime === dateRange.maxTime && query.value.minTime === dateRange.minTime) {
        return;
    }

    const changed = transactionsStore.updateTransactionListFilter({
        dateType: dateRange.dateType,
        maxTime: dateRange.maxTime,
        minTime: dateRange.minTime
    });

    updateUrlWhenChanged(changed);
}

function changeCustomDateFilter(minTime: number, maxTime: number): void {
    if (!minTime || !maxTime) {
        return;
    }

    let dateType: number | null = getDateTypeByBillingCycleDateRange(minTime, maxTime, firstDayOfWeek.value, fiscalYearStart.value, DateRangeScene.Normal, accountsStore.getAccountStatementDate(query.value.accountIds));

    if (!dateType) {
        dateType = getDateTypeByDateRange(minTime, maxTime, firstDayOfWeek.value, fiscalYearStart.value, DateRangeScene.Normal);
    }

    if (pageType.value === TransactionListPageType.Calendar.type) {
        currentCalendarDate.value = getValidMonthDayOrCurrentDayShortDate(minTime, currentCalendarDate.value);
        const dateRange = getFullMonthDateRange(minTime, maxTime, firstDayOfWeek.value, fiscalYearStart.value);

        if (dateRange) {
            minTime = dateRange.minTime;
            maxTime = dateRange.maxTime;
            dateType = dateRange.dateType;
            currentCalendarDate.value = getValidMonthDayOrCurrentDayShortDate(minTime, currentCalendarDate.value);
        }
    }

    if (query.value.dateType === dateType && query.value.maxTime === maxTime && query.value.minTime === minTime) {
        showCustomDateRangeDialog.value = false;
        return;
    }

    const changed = transactionsStore.updateTransactionListFilter({
        dateType: dateType,
        maxTime: maxTime,
        minTime: minTime
    });

    showCustomDateRangeDialog.value = false;
    updateUrlWhenChanged(changed);
}

function changeCustomMonthDateFilter(yearMonth: Year0BasedMonth): void {
    if (!yearMonth) {
        return;
    }

    const minTime = getYearMonthFirstUnixTime(yearMonth);
    const maxTime = getYearMonthLastUnixTime(yearMonth);
    const dateType = getDateTypeByDateRange(minTime, maxTime, firstDayOfWeek.value, fiscalYearStart.value, DateRangeScene.Normal);

    if (pageType.value === TransactionListPageType.Calendar.type) {
        currentCalendarDate.value = getValidMonthDayOrCurrentDayShortDate(minTime, currentCalendarDate.value);
    }

    if (query.value.dateType === dateType && query.value.maxTime === maxTime && query.value.minTime === minTime) {
        showCustomMonthDialog.value = false;
        return;
    }

    const changed = transactionsStore.updateTransactionListFilter({
        dateType: dateType,
        maxTime: maxTime,
        minTime: minTime
    });

    showCustomMonthDialog.value = false;
    updateUrlWhenChanged(changed);
}

function shiftDateRange(startTime: number, endTime: number, scale: number): void {
    if (recentMonthDateRanges.value[recentDateRangeIndex.value]?.dateType === DateRange.All.type) {
        return;
    }

    let newDateRange: TimeRangeAndDateType | null = null;

    if (DateRange.isBillingCycle(query.value.dateType) || query.value.dateType === DateRange.Custom.type) {
        newDateRange = getShiftedDateRangeAndDateTypeForBillingCycle(startTime, endTime, scale, firstDayOfWeek.value, fiscalYearStart.value, DateRangeScene.Normal, accountsStore.getAccountStatementDate(query.value.accountIds));
    }

    if (!newDateRange) {
        newDateRange = getShiftedDateRangeAndDateType(startTime, endTime, scale, firstDayOfWeek.value, fiscalYearStart.value, DateRangeScene.Normal);
    }

    if (pageType.value === TransactionListPageType.Calendar.type) {
        currentCalendarDate.value = getValidMonthDayOrCurrentDayShortDate(newDateRange.minTime, currentCalendarDate.value);
        const fullMonthDateRange = getFullMonthDateRange(newDateRange.minTime, newDateRange.maxTime, firstDayOfWeek.value, fiscalYearStart.value);

        if (fullMonthDateRange) {
            newDateRange = fullMonthDateRange;
            currentCalendarDate.value = getValidMonthDayOrCurrentDayShortDate(newDateRange.minTime, currentCalendarDate.value);
        }
    }

    const changed = transactionsStore.updateTransactionListFilter({
        dateType: newDateRange.dateType,
        maxTime: newDateRange.maxTime,
        minTime: newDateRange.minTime
    });

    updateUrlWhenChanged(changed);
}

function changeTypeFilter(type: number): void {
    let newCategoryFilter: string | undefined = undefined;

    if (type && query.value.categoryIds) {
        newCategoryFilter = '';

        for (const categoryId of keys(queryAllFilterCategoryIds.value)) {
            const category = allCategories.value[categoryId];

            if (category && category.type === transactionTypeToCategoryType(type)) {
                if (newCategoryFilter.length > 0) {
                    newCategoryFilter += ',';
                }

                newCategoryFilter += categoryId;
            }
        }
    }

    const changed = transactionsStore.updateTransactionListFilter({
        type: type,
        categoryIds: newCategoryFilter
    });

    updateUrlWhenChanged(changed);
}

function changeCategoryFilter(categoryIds: string): void {
    categoryMenuState.value = false;

    if (query.value.categoryIds === categoryIds) {
        return;
    }

    const changed = transactionsStore.updateTransactionListFilter({
        categoryIds: categoryIds
    });

    updateUrlWhenChanged(changed);
}

function changeMultipleCategoriesFilter(changed: boolean): void {
    categoryMenuState.value = false;
    showFilterCategoryDialog.value = false;
    updateUrlWhenChanged(changed);
}

function changeAccountFilter(accountIds: string): void {
    if (query.value.accountIds === accountIds) {
        return;
    }

    const changed = transactionsStore.updateTransactionListFilter({
        accountIds: accountIds
    });

    updateUrlWhenChanged(changed);
}

function changeMultipleAccountsFilter(changed: boolean): void {
    showFilterAccountDialog.value = false;
    updateUrlWhenChanged(changed);
}

function changeTagFilter(tagFilter: string): void {
    if (query.value.tagFilter === tagFilter) {
        return;
    }

    const changed = transactionsStore.updateTransactionListFilter({
        tagFilter: tagFilter
    });

    updateUrlWhenChanged(changed);
}

function changeMultipleTagsFilter(changed: boolean): void {
    showFilterTagDialog.value = false;

    updateUrlWhenChanged(changed);
}

function changeKeywordFilter(keyword: string): void {
    if (query.value.keyword === keyword) {
        return;
    }

    const changed = transactionsStore.updateTransactionListFilter({
        keyword: keyword
    });

    updateUrlWhenChanged(changed);
}

function changeAmountFilter(filterType: string): void {
    currentAmountFilterType.value = '';
    amountMenuState.value = false;

    if (query.value.amountFilter === filterType) {
        return;
    }

    let amountFilter = filterType;

    if (filterType) {
        const amountCount = getAmountFilterParameterCount(filterType);

        if (!amountCount) {
            return;
        }

        if (amountCount === 1) {
            amountFilter += ':' + currentAmountFilterValue1.value;
        } else if (amountCount === 2) {
            if (currentAmountFilterValue2.value < currentAmountFilterValue1.value) {
                snackbar.value?.showMessage('Incorrect amount range');
                return;
            }

            amountFilter += ':' + currentAmountFilterValue1.value + ':' + currentAmountFilterValue2.value;
        } else {
            return;
        }
    }

    if (query.value.amountFilter === amountFilter) {
        return;
    }

    const changed = transactionsStore.updateTransactionListFilter({
        amountFilter: amountFilter
    });

    updateUrlWhenChanged(changed);
}

function add(template?: TransactionTemplate): void {
    const currentUnixTime = getCurrentUnixTime();

    let newTransactionTime: number | undefined = undefined;

    if (query.value.maxTime && query.value.minTime) {
        if (query.value.maxTime < currentUnixTime) {
            newTransactionTime = query.value.maxTime;
        } else if (currentUnixTime < query.value.minTime) {
            newTransactionTime = query.value.minTime;
        }
    }

    editDialog.value?.open({
        time: newTransactionTime,
        type: query.value.type,
        categoryId: queryAllFilterCategoryIdsCount.value === 1 ? query.value.categoryIds : '',
        accountId: queryAllFilterAccountIdsCount.value === 1 ? query.value.accountIds : '',
        tagIds: objectFieldWithValueToArrayItem(queryAllFilterTagIds.value, true).join(',') || '',
        template: template
    }).then(result => {
        if (result && result.message) {
            snackbar.value?.showMessage(result.message);
        }

        reload(false, false);
    }).catch(error => {
        if (error) {
            snackbar.value?.showError(error);
        }
    });
}

function addByRecognizingImage(): void {
    aiImageRecognitionDialog.value?.open().then(result => {
        editDialog.value?.open({
            time: result.time,
            type: result.type,
            categoryId: result.categoryId,
            accountId: result.sourceAccountId,
            destinationAccountId: result.destinationAccountId,
            amount: result.sourceAmount,
            destinationAmount: result.destinationAmount,
            tagIds: result.tagIds ? result.tagIds.join(',') : undefined,
            comment: result.comment,
            noTransactionDraft: true
        }).then(result => {
            if (result && result.message) {
                snackbar.value?.showMessage(result.message);
            }

            reload(false, false);
        }).catch(error => {
            if (error) {
                snackbar.value?.showError(error);
            }
        });
    });
}

function addByVoiceInput(): void {
    if (!isSpeechRecognitionSupported()) {
        snackbar.value?.showMessage('Speech recognition is not supported by this browser');
        return;
    }

    recognizingByVoice.value = true;
    snackbar.value?.showMessage('Listening...');

    recognizeTransactionByVoice({
        languageTag: getCurrentLanguageInfo().alternativeLanguageTag,
        allCategories: transactionCategoriesStore.allTransactionCategories,
        allVisibleAccounts: accountsStore.allVisiblePlainAccounts,
        allTags: transactionTagsStore.allTransactionTags,
        recognizeTransactionText: (text: string) => transactionsStore.recognizeTransactionText({ text })
    }).then(result => {
        editDialog.value?.open({
            time: result.time,
            type: result.type,
            categoryId: result.categoryId,
            accountId: result.sourceAccountId,
            destinationAccountId: result.destinationAccountId,
            amount: result.sourceAmount,
            destinationAmount: result.destinationAmount,
            tagIds: result.tagIds ? result.tagIds.join(',') : undefined,
            comment: result.comment,
            noTransactionDraft: true
        }).then(result => {
            if (result && result.message) {
                snackbar.value?.showMessage(result.message);
            }

            reload(false, false);
        }).catch(error => {
            if (error) {
                snackbar.value?.showError(error);
            }
        });
    }).catch(error => {
        snackbar.value?.showError(error);
    }).finally(() => {
        recognizingByVoice.value = false;
    });
}

function importTransaction(): void {
    importDialog.value?.open().then(() => {
        reload(false, false);
    }).catch(error => {
        if (error) {
            snackbar.value?.showError(error);
        }
    });
}

function exportTransactions(fileExtension: string): void {
    if (exportingData.value) {
        return;
    }

    const nickname = userStore.currentUserNickname;
    let exportFileName = '';

    if (nickname) {
        exportFileName = tt('dataExport.exportFilename', {
            nickname: nickname
        }) + '.' + fileExtension;
    } else {
        exportFileName = tt('dataExport.defaultExportFilename') + '.' + fileExtension;
    }

    const exportTransactionReq = transactionsStore.getExportTransactionDataRequestByTransactionFilter();

    exportingData.value = true;

    userStore.getExportedUserData(fileExtension, exportTransactionReq).then(data => {
        startDownloadFile(exportFileName, data);
        exportingData.value = false;
    }).catch(error => {
        exportingData.value = false;

        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

function show(transaction: Transaction): void {
    editDialog.value?.open({
        id: transaction.id,
        currentTransaction: transaction
    }).then(result => {
        if (result && result.message) {
            snackbar.value?.showMessage(result.message);
        }

        reload(false, false);
    }).catch(error => {
        if (error) {
            snackbar.value?.showError(error);
        }
    });
}

function scrollTimeMenuToSelectedItem(opened: boolean): void {
    if (opened) {
        scrollMenuToSelectedItem(timeFilterMenu.value);
    }
}

function scrollCategoryMenuToSelectedItem(opened: boolean): void {
    if (opened) {
        scrollMenuToSelectedItem(categoryFilterMenu.value);
    }
}

function scrollAmountMenuToSelectedItem(opened: boolean): void {
    if (opened) {
        currentAmountFilterType.value = '';

        let amount1 = 0, amount2 = 0;

        if (isString(query.value.amountFilter)) {
            try {
                const filterItems = query.value.amountFilter.split(':');
                const amountCount = getAmountFilterParameterCount(filterItems[0] as string);

                if (filterItems.length === 2 && amountCount === 1) {
                    amount1 = parseInt(filterItems[1] as string);
                } else if (filterItems.length === 3 && amountCount === 2) {
                    amount1 = parseInt(filterItems[1] as string);
                    amount2 = parseInt(filterItems[2] as string);
                }
            } catch (ex) {
                logger.warn('cannot parse amount from filter value, original value is ' + query.value.amountFilter, ex);
            }
        }

        currentAmountFilterValue1.value = amount1;
        currentAmountFilterValue2.value = amount2;

        scrollMenuToSelectedItem(amountFilterMenu.value);
    }
}

function scrollAccountMenuToSelectedItem(opened: boolean): void {
    if (opened) {
        scrollMenuToSelectedItem(accountFilterMenu.value);
    }
}

function scrollTagMenuToSelectedItem(opened: boolean): void {
    if (opened) {
        scrollMenuToSelectedItem(tagFilterMenu.value);
    }
}

function scrollMenuToSelectedItem(menu: VMenu | null): void {
    nextTick(() => {
        scrollToSelectedItem(menu?.contentEl, 'div.v-list', 'div.v-list', 'div.v-list-item.list-item-selected');
    });
}

function onShowDateRangeError(message: string): void {
    snackbar.value?.showError(message);
}

onBeforeRouteUpdate((to) => {
    if (to.query) {
        init({
            initPageType: (to.query['pageType'] as string | null) || '0',
            initDateType: (to.query['dateType'] as string | null) || undefined,
            initMinTime: (to.query['minTime'] as string | null) || undefined,
            initMaxTime: (to.query['maxTime'] as string | null) || undefined,
            initType: (to.query['type'] as string | null) || undefined,
            initCategoryIds: (to.query['categoryIds'] as string | null) || undefined,
            initAccountIds: (to.query['accountIds'] as string | null) || undefined,
            initTagFilter: (to.query['tagFilter'] as string | null) || undefined,
            initAmountFilter: (to.query['amountFilter'] as string | null) || undefined,
            initKeyword: (to.query['keyword'] as string | null) || undefined
        });
    } else {
        init({});
    }
});

watch(() => desktopPageStore.showAddTransactionDialogInTransactionList, (newValue) => {
    if (newValue) {
        desktopPageStore.resetShowAddTransactionDialogInTransactionList();
        add();
    }
});

init(props);
</script>

<style>
.transaction-keyword-filter .v-input--density-compact {
    --v-input-control-height: 38px !important;
    --v-input-padding-top: 5px !important;
    --v-input-padding-bottom: 5px !important;
    --v-input-chips-margin-top: 0px !important;
    --v-input-chips-margin-bottom: 0px !important;
    inline-size: 100%;

    .v-field__input {
        min-block-size: 38px !important;
    }
}

.transaction-list-datetime-range {
    min-height: 28px;
    flex-wrap: wrap;
    row-gap: 1rem;
}

.transaction-list-custom-datetime-range {
    line-height: 1rem;
}

.transaction-list-datetime-range .transaction-list-datetime-range-text {
    color: rgba(var(--v-theme-on-background), var(--v-medium-emphasis-opacity)) !important;
}

.v-table.transaction-table > .v-table__wrapper > table {
    table-layout: auto;

    th, td {
        white-space: nowrap;
    }

    .transaction-table-column-description {
        width: 100%;
        max-width: 0;
        min-width: 220px;
    }
}

.v-table.transaction-table .transaction-list-row-date > td {
    height: 40px !important;
}

.transaction-table .transaction-table-column-time {
    min-width: 110px;
}

.transaction-table .transaction-table-column-category {
    min-width: 140px;
}

.transaction-table .transaction-table-column-amount {
    min-width: 120px;
}

.transaction-table .transaction-table-column-account {
    min-width: 160px;
}

.transaction-table .transaction-table-column-tags {
    min-width: 90px;
}

.transaction-table .transaction-table-column-category .v-btn,
.transaction-table .transaction-table-column-account .v-btn {
    font-size: 0.75rem;
}

.transaction-table .transaction-table-column-category .v-btn .v-btn__append,
.transaction-table .transaction-table-column-account .v-btn .v-btn__append {
    margin-inline-start: 0in;
}

.transaction-table .transaction-table-column-tags .v-chip.transaction-tag {
    margin-inline-end: 4px;
    margin-top: 2px;
    margin-bottom: 2px;
}

.transaction-table .transaction-table-column-tags .v-chip.transaction-tag > .v-chip__content {
    display: block;
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
}

.transaction-time-menu .item-icon,
.transaction-category-menu .item-icon,
.transaction-amount-menu .item-icon,
.transaction-account-menu .item-icon,
.transaction-tag-menu .item-icon,
.transaction-table .item-icon {
    padding-bottom: 3px;
}

.transaction-amount-filter-value {
    width: 100px;
}

.transaction-amount-filter-value input.v-field__input {
    min-height: 32px !important;
    padding: 0 8px 0 8px;
}

.transaction-category-menu .has-children-item-selected span,
.transaction-category-menu .item-in-multiple-selection span,
.transaction-account-menu .item-in-multiple-selection span,
.transaction-tag-menu .item-in-multiple-selection span {
    font-weight: bold;
}

.transaction-calendar-container .dp__main .dp__menu {
    --dp-border-radius: 6px;
    --dp-menu-border-color: rgba(var(--v-border-color), var(--v-border-opacity));
}

.transaction-calendar-container .dp__main .dp__calendar {
    --dp-border-color: rgba(var(--v-border-color), var(--v-border-opacity));
}

.transaction-calendar-container .dp__main .dp__calendar .dp__calendar_row {
    --dp-cell-size: 80px;
    --dp-primary-color: rgba(var(--v-theme-primary), var(--v-activated-opacity));
    --dp-primary-text-color: rgb(var(--v-theme-primary));
}

.transaction-calendar-container .dp__main.transaction-calendar-with-alternate-date .dp__calendar .dp__calendar_row {
    --dp-cell-size: 100px;
}

.transaction-calendar-container .dp__main .dp__calendar .dp__calendar_row > .dp__calendar_item {
    overflow: hidden;
}

.transaction-calendar-container .dp__main .dp__calendar .dp__calendar_row > .dp__calendar_item .transaction-calendar-daily-amounts > span.transaction-calendar-alternate-date {
    font-size: 0.9rem;
}

.transaction-calendar-container .dp__main .dp__calendar .dp__calendar_row > .dp__calendar_item .transaction-calendar-daily-amounts > span.transaction-calendar-daily-amount {
    font-size: 0.95rem;
}
</style>

<style scoped>
.transaction-category-scroll {
    max-width: 100%;
    overflow-x: auto;
    overscroll-behavior-x: contain;
}

.transaction-category-grid {
    display: grid;
    width: max-content;
    min-width: 100%;
    align-items: start;
}

.transaction-category-heading {
    display: flex;
    align-items: center;
    gap: 10px;
    height: 48px;
    padding: 0 20px;
    font-size: 13px;
    border-bottom: 1px solid rgba(var(--v-theme-on-surface), 0.08);
}

.transaction-category-date {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 8px;
    padding: 10px 20px;
    font-size: 12px;
    color: rgba(var(--v-theme-on-surface), 0.6);
    background: rgba(var(--v-theme-on-surface), 0.025);
    border-block: 1px solid rgba(var(--v-theme-on-surface), 0.06);
}

.transaction-daily-total {
    display: inline-flex;
    align-items: baseline;
    gap: 6px;
    margin-inline-start: 12px;
    white-space: nowrap;
    font-variant-numeric: tabular-nums;
}

.transaction-category-cell {
    min-width: 0;
    border-right: 1px solid rgba(var(--v-theme-on-surface), 0.06);
}

.transaction-category-list {
    list-style: none;
    margin: 0;
    padding: 0;
}

.transaction-category-item {
    display: flex;
    flex-direction: column;
    justify-content: center;
    gap: 6px;
    width: 100%;
    height: 76px;
    padding: 12px 20px;
    text-align: start;
    font-size: 13px;
    border-bottom: 1px solid rgba(var(--v-theme-on-surface), 0.06);
}

.transaction-category-item:hover {
    background: rgba(var(--v-theme-on-surface), 0.04);
}

.transaction-category-item:focus-visible {
    outline: 2px solid rgb(var(--v-theme-primary));
    outline-offset: -2px;
}

.transaction-category-description {
    display: block;
    max-width: 100%;
}

.transaction-category-details {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    width: 100%;
    font-variant-numeric: tabular-nums;
}

.transaction-category-time {
    color: rgba(var(--v-theme-on-surface), 0.55);
    font-size: 12px;
    text-align: end;
}

.transaction-category-loading {
    opacity: 0.5;
}

.transaction-workspace {
    border: none;
    border-radius: 16px !important;
    background: rgb(var(--v-theme-surface)) !important;
    box-shadow: none !important;
}

.transaction-content {
    min-height: 640px;
}

.transaction-content :deep(.v-card-item) {
    padding: 24px 24px 16px;
}

.transaction-content :deep(.v-card-title) {
    padding: 0;
}

.transaction-toolbar {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 12px;
}

.transaction-view-switch {
    height: 36px !important;
    block-size: 36px !important;
    padding: 3px !important;
    border: none !important;
    border-radius: 10px;
    background: rgba(var(--v-theme-on-surface), 0.055);
    flex-shrink: 0;
}

.transaction-view-switch :deep(.v-btn) {
    height: 30px !important;
    block-size: 30px !important;
    inline-size: auto !important;
    border: none;
    border-radius: 7px !important;
    padding-inline: 16px;
    color: rgba(var(--v-theme-on-surface), 0.65);
}

.transaction-view-switch :deep(.v-btn--active) {
    color: rgb(var(--v-theme-on-surface));
    background: rgb(var(--v-theme-surface));
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.12);
}

.transaction-view-switch :deep(.v-btn--active > .v-btn__overlay) {
    opacity: 0;
}

.transaction-keyword-filter {
    width: 280px;
    min-width: 180px;
    margin-inline-start: auto;
}

.transaction-keyword-filter :deep(.v-field) {
    border-radius: 10px;
    background: rgba(var(--v-theme-on-surface), 0.045);
}

.transaction-add-button {
    border-radius: 10px;
    box-shadow: none;
}

.transaction-list-datetime-range {
    gap: 12px;
    padding-block: 4px;
}

.transaction-date-control {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 2px;
}

.transaction-totals {
    flex-wrap: wrap;
    font-size: 13px;
    font-variant-numeric: tabular-nums;
}

.transaction-totals .text-subtitle-1,
.transaction-custom-range {
    font-size: 13px !important;
    color: rgba(var(--v-theme-on-surface), 0.55);
}

.transaction-table :deep(th) {
    height: 42px !important;
    font-size: 12px;
    color: rgba(var(--v-theme-on-surface), 0.5) !important;
    background: transparent !important;
}

.transaction-table :deep(td),
.transaction-table :deep(th) {
    padding-inline: 24px !important;
    border-bottom-color: rgba(var(--v-theme-on-surface), 0.06) !important;
}

.transaction-table :deep(.transaction-table-row-data td) {
    height: 56px;
    font-variant-numeric: tabular-nums;
}

.transaction-table :deep(.transaction-list-row-date td) {
    background: rgba(var(--v-theme-on-surface), 0.025);
    font-size: 12px;
    font-weight: 500 !important;
    color: rgba(var(--v-theme-on-surface), 0.6);
}

.transaction-table :deep(.transaction-list-row-date .v-chip) {
    background: transparent;
    font-weight: 400;
}

.transaction-table :deep(.transaction-table-column-time) {
    color: rgba(var(--v-theme-on-surface), 0.55);
}

@media (max-width: 960px) {
    .transaction-keyword-filter {
        order: 1;
        width: 100%;
    }

    .transaction-view-switch {
        margin-inline-end: auto;
    }
}

@media (max-width: 600px) {
    .transaction-content :deep(.v-card-item) {
        padding: 16px;
    }

    .transaction-toolbar {
        gap: 6px;
    }

    .transaction-view-switch :deep(.v-btn) {
        padding-inline: 10px;
    }

    .transaction-table :deep(td),
    .transaction-table :deep(th) {
        padding-inline: 16px !important;
    }
}
</style>
