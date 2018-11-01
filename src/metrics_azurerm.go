package main

import (
	"fmt"
	"time"
	"sync"
	"context"
	"strconv"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/advisor/mgmt/advisor"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/containerregistry/mgmt/containerregistry"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/network/mgmt/network"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/subscriptions"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/storage/mgmt/storage"
	"github.com/Azure/azure-sdk-for-go/profiles/preview/preview/security/mgmt/security"
	"github.com/Azure/go-autorest/autorest"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	prometheusSubscription *prometheus.GaugeVec
	prometheusResourceGroup *prometheus.GaugeVec
	prometheusVm *prometheus.GaugeVec
	prometheusVmOs *prometheus.GaugeVec
	prometheusPublicIp *prometheus.GaugeVec
	prometheusApiQuota *prometheus.GaugeVec
	prometheusQuota *prometheus.GaugeVec
	prometheusQuotaCurrent *prometheus.GaugeVec
	prometheusQuotaLimit *prometheus.GaugeVec
	prometheusContainerRegistry *prometheus.GaugeVec
	prometheusContainerRegistryQuotaCurrent *prometheus.GaugeVec
	prometheusContainerRegistryQuotaLimit *prometheus.GaugeVec

	// compliance
	prometheusSecuritycenterCompliance *prometheus.GaugeVec
	prometheusAdvisorRecommendations *prometheus.GaugeVec
)

// Create and setup metrics and collection
func initMetricsAzureRm() {
	prometheusSubscription = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_subscription_info",
			Help: "Azure ResourceManager subscription",
		},
		[]string{"resourceID", "subscriptionID", "subscriptionName", "spendingLimit", "quotaID", "locationPlacementID"},
	)

	prometheusResourceGroup = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_resourcegroup_info",
			Help: "Azure ResourceManager resourcegroups",
		},
		append(
			[]string{"resourceID", "subscriptionID", "resourceGroup", "location"},
			prefixSlice(AZURE_RESOURCE_TAG_PREFIX, opts.AzureResourceTags)...
		),
	)

	prometheusVm = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_vm_info",
			Help: "Azure ResourceManager VMs",
		},
		append(
			[]string{"resourceID", "subscriptionID", "location", "resourceGroup", "vmID", "vmName", "vmType", "vmSize", "vmProvisioningState"},
			prefixSlice(AZURE_RESOURCE_TAG_PREFIX, opts.AzureResourceTags)...
		),
	)

	prometheusVmOs = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_vm_os",
			Help: "Azure ResourceManager VM OS",
		},
		[]string{"vmID", "imagePublisher", "imageSku", "imageOffer", "imageVersion"},
	)

	prometheusPublicIp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_publicip_info",
			Help: "Azure ResourceManager public ip",
		},
		append(
			[]string{"resourceID", "subscriptionID", "resourceGroup", "location", "ipAddress", "ipAllocationMethod", "ipAdressVersion"},
			prefixSlice(AZURE_RESOURCE_TAG_PREFIX, opts.AzureResourceTags)...
		),
	)

	prometheusApiQuota = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_ratelimit",
			Help: "Azure ResourceManager ratelimit",
		},
		[]string{"subscriptionID", "scope", "type"},
	)

	prometheusQuota = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_quota_info",
			Help: "Azure ResourceManager quota info",
		},
		[]string{"subscriptionID", "location", "scope", "quota", "quotaName"},
	)

	prometheusQuotaCurrent = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_quota_current",
			Help: "Azure ResourceManager quota current value",
		},
		[]string{"subscriptionID", "location", "scope", "quota"},
	)

	prometheusQuotaLimit = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_quota_limit",
			Help: "Azure ResourceManager quota limit",
		},
		[]string{"subscriptionID", "location", "scope", "quota"},
	)

	prometheusContainerRegistry = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_containerregistry_info",
			Help: "Azure ContainerRegistry limit",
		},
		append(
			[]string{"resourceID", "subscriptionID", "location", "registryName", "resourceGroup", "adminUserEnabled", "skuName", "skuTier"},
			prefixSlice(AZURE_RESOURCE_TAG_PREFIX, opts.AzureResourceTags)...
		),
	)

	prometheusContainerRegistryQuotaCurrent = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_containerregistry_quota_current",
			Help: "Azure ContainerRegistry quota current",
		},
		[]string{"subscriptionID", "registryName", "quotaName", "quotaUnit"},
	)

	prometheusContainerRegistryQuotaLimit = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_containerregistry_quota_limit",
			Help: "Azure ContainerRegistry quota limit",
		},
		[]string{"subscriptionID", "registryName", "quotaName", "quotaUnit"},
	)

	prometheusSecuritycenterCompliance = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_securitycenter_compliance",
			Help: "Azure Audit SecurityCenter compliance status",
		},
		[]string{"subscriptionID", "assessmentType"},
	)

	prometheusAdvisorRecommendations = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azurerm_advisor_recommendation",
			Help: "Azure Audit Advisor recommendation",
		},
		[]string{"subscriptionID", "category", "resourceType", "resourceName", "resourceGroup", "impact", "risk"},
	)

	prometheus.MustRegister(prometheusSubscription)
	prometheus.MustRegister(prometheusResourceGroup)
	prometheus.MustRegister(prometheusVm)
	prometheus.MustRegister(prometheusVmOs)
	prometheus.MustRegister(prometheusPublicIp)
	prometheus.MustRegister(prometheusApiQuota)
	prometheus.MustRegister(prometheusQuota)
	prometheus.MustRegister(prometheusQuotaCurrent)
	prometheus.MustRegister(prometheusQuotaLimit)
	prometheus.MustRegister(prometheusContainerRegistry)
	prometheus.MustRegister(prometheusContainerRegistryQuotaCurrent)
	prometheus.MustRegister(prometheusContainerRegistryQuotaLimit)
	prometheus.MustRegister(prometheusSecuritycenterCompliance)
	prometheus.MustRegister(prometheusAdvisorRecommendations)
}

// Start backgrounded metrics collection
func startMetricsCollectionAzureRm() {
	go func() {
		for {
			go func() {
				runMetricsCollectionAzureRm()
			}()

			Logger.Messsage("run: sleeping %v", opts.ScrapeTime.String())
			time.Sleep(opts.ScrapeTime)
		}
	}()
}

// Metrics run
func runMetricsCollectionAzureRm() {
	var wg sync.WaitGroup
	context := context.Background()

	publicIpChannel := make(chan []string)
	callbackChannel := make(chan func())

	for _, subscription := range AzureSubscriptions {
		Logger.Messsage(
			"subscription[%v]: starting metrics collection",
			*subscription.SubscriptionID,
		)

		// Subscription
		wg.Add(1)
		go func(subscriptionId string) {
			defer wg.Done()
			collectAzureSubscription(context, subscriptionId, callbackChannel)
			Logger.Verbose("subscription[%v]: finished Azure Subscription collection", subscriptionId)
		}(*subscription.SubscriptionID)

		// ResourceGroups
		wg.Add(1)
		go func(subscriptionId string) {
			defer wg.Done()
			collectAzureResourceGroup(context, subscriptionId, callbackChannel)
			Logger.Verbose("subscription[%v]: finished Azure ResourceGroup collection", subscriptionId)
		}(*subscription.SubscriptionID)

		// VMs
		wg.Add(1)
		go func(subscriptionId string) {
			defer wg.Done()
			collectAzureVm(context, subscriptionId, callbackChannel)
			Logger.Verbose("subscription[%v]: finished Azure VirtualMachine collection", subscriptionId)
		}(*subscription.SubscriptionID)

		// Public IPs
		wg.Add(1)
		go func(subscriptionId string) {
			defer wg.Done()
			publicIpChannel <- collectAzurePublicIp(context, subscriptionId, callbackChannel)
			Logger.Verbose("subscription[%v]: finished Azure PublicIP collection", subscriptionId)
		}(*subscription.SubscriptionID)

		// Compute usage
		wg.Add(1)
		go func(subscriptionId string) {
			defer wg.Done()
			collectAzureComputeUsage(context, subscriptionId, callbackChannel)
			Logger.Verbose("subscription[%v]: finished Azure ComputerUsage collection", subscriptionId)
		}(*subscription.SubscriptionID)

		// Network usage
		wg.Add(1)
		go func(subscriptionId string) {
			defer wg.Done()
			// disabled due to
			// https://github.com/Azure/azure-sdk-for-go/issues/2340
			// https://github.com/Azure/azure-rest-api-specs/issues/1624
			//collectAzureNetworkUsage(context, subscriptionId, callbackChannel)
			Logger.Verbose("subscription[%v]: finished Azure NetworkUsage collection (DISABLED -> AZURE BUG)", subscriptionId)
		}(*subscription.SubscriptionID)

		// Storage usage
		wg.Add(1)
		go func(subscriptionId string) {
			defer wg.Done()
			collectAzureStorageUsage(context, subscriptionId, callbackChannel)
			Logger.Verbose("subscription[%v]: finished Azure StorageUsage collection", subscriptionId)
		}(*subscription.SubscriptionID)

		// ContainerRegistries usage
		wg.Add(1)
		go func(subscriptionId string) {
			defer wg.Done()
			collectAzureContainerRegistries(context, subscriptionId, callbackChannel)
			Logger.Verbose("subscription[%v]: finished Azure ContainerRegistries collection", subscriptionId)
		}(*subscription.SubscriptionID)

		// SecurityCompliance
		for _, location := range opts.AzureLocation {
			wg.Add(1)
			go func(subscriptionId, location string) {
				defer wg.Done()
				collectAzureSecurityCompliance(context, subscriptionId, location, callbackChannel)
				Logger.Verbose("subscription[%v]: finished Azure SecurityCompliance collection (%v)", subscriptionId, location)
			}(*subscription.SubscriptionID, location)
		}

		// AdvisorRecommendations
		wg.Add(1)
		go func(subscriptionId string) {
			defer wg.Done()
			collectAzureAdvisorRecommendations(context, subscriptionId, callbackChannel)
			Logger.Verbose("subscription[%v]: finished Azure AdvisorRecommendations collection", subscriptionId)
		}(*subscription.SubscriptionID)
	}

	// process publicIP list and pass it to portscanner
	go func() {
		publicIpList := []string{}
		for ipAddressList := range publicIpChannel {
			publicIpList = append(publicIpList, ipAddressList...)
		}

		// update portscanner public ips
		if portscanner != nil {
			portscanner.SetIps(publicIpList)
			portscanner.Cleanup()
			portscanner.Enable()
		}
		Logger.Messsage("run: collected %v public IPs", len(publicIpList))
	}()

	// collect metrics (callbacks) and proceses them
	go func() {
		var callbackList []func()
		for callback := range callbackChannel {
			callbackList = append(callbackList, callback)
		}

		prometheusSubscription.Reset()
		prometheusResourceGroup.Reset()
		prometheusVm.Reset()
		prometheusVmOs.Reset()
		prometheusPublicIp.Reset()
		prometheusApiQuota.Reset()
		prometheusQuota.Reset()
		prometheusQuotaCurrent.Reset()
		prometheusQuotaLimit.Reset()
		prometheusContainerRegistry.Reset()
		prometheusContainerRegistryQuotaCurrent.Reset()
		prometheusContainerRegistryQuotaLimit.Reset()
		prometheusSecuritycenterCompliance.Reset()
		prometheusAdvisorRecommendations.Reset()
		for _, callback := range callbackList {
			callback()
		}

		Logger.Messsage("run: finished")
	}()

	// wait for all funcs
	wg.Wait()
	close(publicIpChannel)
	close(callbackChannel)
}

// Collect Azure Subscription metrics
func collectAzureSubscription(context context.Context, subscriptionId string, callback chan<- func()) {
	subscriptionClient := subscriptions.NewClient()
	subscriptionClient.Authorizer = AzureAuthorizer

	sub, err := subscriptionClient.Get(context, subscriptionId)
	if err != nil {
		panic(err)
	}

	callback <- func() {
		prometheusSubscription.With(
			prometheus.Labels{
				"resourceID": *sub.ID,
				"subscriptionID":      *sub.SubscriptionID,
				"subscriptionName":    *sub.DisplayName,
				"spendingLimit":       string(sub.SubscriptionPolicies.SpendingLimit),
				"quotaID":             *sub.SubscriptionPolicies.QuotaID,
				"locationPlacementID": *sub.SubscriptionPolicies.LocationPlacementID,
			},
		).Set(1)
	}

	// subscription rate limits
	probeProcessHeader(sub.Response, "x-ms-ratelimit-remaining-subscription-reads", prometheus.Labels{"subscriptionID": subscriptionId, "scope": "subscription", "type": "read"}, callback)
	probeProcessHeader(sub.Response, "x-ms-ratelimit-remaining-subscription-resource-requests", prometheus.Labels{"subscriptionID": subscriptionId, "scope": "subscription", "type": "resource-requests"}, callback)
	probeProcessHeader(sub.Response, "x-ms-ratelimit-remaining-subscription-resource-entities-read", prometheus.Labels{"subscriptionID": subscriptionId, "scope": "subscription", "type": "resource-entities-read"}, callback)

	// tenant rate limits
	probeProcessHeader(sub.Response, "x-ms-ratelimit-remaining-tenant-reads", prometheus.Labels{"subscriptionID": subscriptionId, "scope": "tenant", "type": "read"}, callback)
	probeProcessHeader(sub.Response, "x-ms-ratelimit-remaining-tenant-resource-requests", prometheus.Labels{"subscriptionID": subscriptionId, "scope": "tenant", "type": "resource-requests"}, callback)
	probeProcessHeader(sub.Response, "x-ms-ratelimit-remaining-tenant-resource-entities-read", prometheus.Labels{"subscriptionID": subscriptionId, "scope": "tenant", "type": "resource-entities-read"}, callback)
}

func addAzureResourceTags(tags map[string]*string, labels prometheus.Labels) (prometheus.Labels) {
	for _, rgTag := range opts.AzureResourceTags {
		rgTabLabel := AZURE_RESOURCE_TAG_PREFIX + rgTag

		if _, ok := tags[rgTag]; ok {
			labels[rgTabLabel] = *tags[rgTag]
		} else {
			labels[rgTabLabel] = ""
		}
	}

	return labels
}

// Collect Azure ResourceGroup metrics
func collectAzureResourceGroup(context context.Context, subscriptionId string, callback chan<- func()) {
	resourceGroupClient := resources.NewGroupsClient(subscriptionId)
	resourceGroupClient.Authorizer = AzureAuthorizer

	resourceGroupResult, err := resourceGroupClient.ListComplete(context, "", nil)
	if err != nil {
		panic(err)
	}

	for _, item := range *resourceGroupResult.Response().Value {
		infoLabels := prometheus.Labels{
			"resourceID": *item.ID,
			"subscriptionID": subscriptionId,
			"resourceGroup": *item.Name,
			"location": *item.Location,
		}
		infoLabels = addAzureResourceTags(item.Tags, infoLabels)

		callback <- func() {
			prometheusResourceGroup.With(infoLabels).Set(1)
		}
	}
}

// Collect Azure PublicIP metrics
func collectAzurePublicIp(context context.Context, subscriptionId string, callback chan<- func()) (ipAddressList []string) {
	netPublicIpClient := network.NewPublicIPAddressesClient(subscriptionId)
	netPublicIpClient.Authorizer = AzureAuthorizer

	list, err := netPublicIpClient.ListAll(context)
	if err != nil {
		panic(err)
	}

	for _, val := range list.Values() {
		location := *val.Location
		ipAddress := ""
		ipAllocationMethod := string(val.PublicIPAllocationMethod)
		ipAdressVersion := string(val.PublicIPAddressVersion)
		gaugeValue := float64(1)

		if val.IPAddress != nil {
			ipAddress = *val.IPAddress
			ipAddressList = append(ipAddressList, ipAddress)
		} else {
			ipAddress = "not allocated"
			gaugeValue = 0
		}

		infoLabels := prometheus.Labels{
			"resourceID": *val.ID,
			"subscriptionID":     subscriptionId,
			"resourceGroup":      extractResourceGroupFromAzureId(*val.ID),
			"location":           location,
			"ipAddress":          ipAddress,
			"ipAllocationMethod": ipAllocationMethod,
			"ipAdressVersion":    ipAdressVersion,
		}
		infoLabels = addAzureResourceTags(val.Tags, infoLabels)


		callback <- func() {
			prometheusPublicIp.With(infoLabels).Set(gaugeValue)
		}
	}

	return
}

// Collect Azure ComputeUsage metrics
func collectAzureComputeUsage(context context.Context, subscriptionId string, callback chan<- func()) {
	computeClient := compute.NewUsageClient(subscriptionId)
	computeClient.Authorizer = AzureAuthorizer

	for _, location := range opts.AzureLocation {
		list, err := computeClient.List(context, location)

		if err != nil {
			panic(err)
		}

		for _, val := range list.Values() {
			quotaName := *val.Name.Value
			quotaNameLocalized := *val.Name.LocalizedValue
			currentValue := float64(*val.CurrentValue)
			limitValue := float64(*val.Limit)

			labels := prometheus.Labels{
				"subscriptionID": subscriptionId,
				"location": location,
				"scope": "compute",
				"quota": quotaName,
			}

			infoLabels := prometheus.Labels{
				"subscriptionID": subscriptionId,
				"location": location,
				"scope": "compute",
				"quota": quotaName,
				"quotaName": quotaNameLocalized,
			}

			callback <- func() {
				prometheusQuota.With(infoLabels).Set(1)
				prometheusQuotaCurrent.With(labels).Set(currentValue)
				prometheusQuotaLimit.With(labels).Set(limitValue)
			}
		}
	}
}

// Collect Azure NetworkUsage metrics
func collectAzureNetworkUsage(context context.Context, subscriptionId string, callback chan<- func()) {
	networkClient := network.NewUsagesClient(subscriptionId)
	networkClient.Authorizer = AzureAuthorizer

	for _, location := range opts.AzureLocation {
		list, err := networkClient.List(context, location)

		if err != nil {
			panic(err)
		}

		for _, val := range list.Values() {
			quotaName := *val.Name.Value
			quotaNameLocalized := *val.Name.LocalizedValue
			currentValue := float64(*val.CurrentValue)
			limitValue := float64(*val.Limit)

			labels := prometheus.Labels{
				"subscriptionID": subscriptionId,
				"location": location,
				"scope": "storage",
				"quota": quotaName,
			}

			infoLabels := prometheus.Labels{
				"subscriptionID": subscriptionId,
				"location": location,
				"scope": "storage",
				"quota": quotaName,
				"quotaName": quotaNameLocalized,
			}

			callback <- func() {
				prometheusQuota.With(infoLabels).Set(1)
				prometheusQuotaCurrent.With(labels).Set(currentValue)
				prometheusQuotaLimit.With(labels).Set(limitValue)
			}
		}
	}
}

// Collect Azure StorageUsage metrics
func collectAzureStorageUsage(context context.Context, subscriptionId string, callback chan<- func()) {
	storageClient := storage.NewUsageClient(subscriptionId)
	storageClient.Authorizer = AzureAuthorizer

	for _, location := range opts.AzureLocation {
		list, err := storageClient.List(context)

		if err != nil {
			panic(err)
		}

		for _, val := range *list.Value {
			quotaName := *val.Name.Value
			quotaNameLocalized := *val.Name.LocalizedValue
			currentValue := float64(*val.CurrentValue)
			limitValue := float64(*val.Limit)

			labels := prometheus.Labels{
				"subscriptionID": subscriptionId,
				"location": location,
				"scope": "storage",
				"quota": quotaName,
			}

			infoLabels := prometheus.Labels{
				"subscriptionID": subscriptionId,
				"location": location,
				"scope": "storage",
				"quota": quotaName,
				"quotaName": quotaNameLocalized,
			}

			callback <- func() {
				prometheusQuota.With(infoLabels).Set(1)
				prometheusQuotaCurrent.With(labels).Set(currentValue)
				prometheusQuotaLimit.With(labels).Set(limitValue)
			}
		}
	}
}

func collectAzureVm(context context.Context, subscriptionId string, callback chan<- func()) {
	computeClient := compute.NewVirtualMachinesClient(subscriptionId)
	computeClient.Authorizer = AzureAuthorizer

	list, err := computeClient.ListAllComplete(context)

	if err != nil {
		panic(err)
	}


	for list.NotDone() {
		val := list.Value()

		infoLabels := prometheus.Labels{
			"resourceID": *val.ID,
			"subscriptionID": subscriptionId,
			"location": *val.Location,
			"resourceGroup": extractResourceGroupFromAzureId(*val.ID),
			"vmID": *val.VMID,
			"vmName": *val.Name,
			"vmType": *val.Type,
			"vmSize": string(val.VirtualMachineProperties.HardwareProfile.VMSize),
			"vmProvisioningState": *val.ProvisioningState,
		}
		infoLabels = addAzureResourceTags(val.Tags, infoLabels)

		osLabels := prometheus.Labels{
			"vmID": *val.VMID,
			"imagePublisher": *val.StorageProfile.ImageReference.Publisher,
			"imageSku": *val.StorageProfile.ImageReference.Sku,
			"imageOffer": *val.StorageProfile.ImageReference.Offer,
			"imageVersion": *val.StorageProfile.ImageReference.Version,
		}

		callback <- func() {
			prometheusVm.With(infoLabels).Set(1)
			prometheusVmOs.With(osLabels).Set(1)
		}

		if list.Next() != nil {
			break
		}
	}
}


func collectAzureContainerRegistries(context context.Context, subscriptionId string, callback chan<- func()) {
	acrClient := containerregistry.NewRegistriesClient(subscriptionId)
	acrClient.Authorizer = AzureAuthorizer

	list, err := acrClient.ListComplete(context)

	if err != nil {
		panic(err)
	}


	for list.NotDone() {
		val := list.Value()

		arcUsage, err := acrClient.ListUsages(context, extractResourceGroupFromAzureId(*val.ID), *val.Name)

		if err != nil {
			ErrorLogger.Error(fmt.Sprintf("subscription[%v]: unable to fetch ACR usage for %v", subscriptionId, *val.Name), err)
		}

		skuName := ""
		skuTier := ""

		if val.Sku != nil {
			skuName = string(val.Sku.Name)
			skuTier = string(val.Sku.Tier)
		}

		infoLabels := prometheus.Labels{
			"resourceID": *val.ID,
			"subscriptionID": subscriptionId,
			"location": *val.Location,
			"registryName": *val.Name,
			"resourceGroup": extractResourceGroupFromAzureId(*val.ID),
			"adminUserEnabled": boolToString(*val.AdminUserEnabled),
			"skuName": skuName,
			"skuTier": skuTier,
		}
		infoLabels = addAzureResourceTags(val.Tags, infoLabels)

		callback <- func() {
			prometheusContainerRegistry.With(infoLabels).Set(1)

			if arcUsage.Value != nil {
				for _, usage := range *arcUsage.Value {
					quotaLabels := prometheus.Labels{
						"subscriptionID": subscriptionId,
						"registryName": *val.Name,
						"quotaUnit": string(usage.Unit),
						"quotaName": *usage.Name,
					}

					prometheusContainerRegistryQuotaCurrent.With(quotaLabels).Set(float64(*usage.CurrentValue))
					prometheusContainerRegistryQuotaLimit.With(quotaLabels).Set(float64(*usage.Limit))
				}
			}
		}

		if list.Next() != nil {
			break
		}
	}
}


// read header and set prometheus api quota (if found)
func probeProcessHeader(response autorest.Response, header string, labels prometheus.Labels, callback chan<- func()) {
	if val := response.Header.Get(header); val != "" {
		valFloat, err := strconv.ParseFloat(val, 64)

		if err == nil {
			callback <- func() {
				prometheusApiQuota.With(labels).Set(valFloat)
			}
		} else {
			ErrorLogger.Error(fmt.Sprintf("Failed to parse value '%v':", val), err)
		}
	}
}

func collectAzureSecurityCompliance(context context.Context, subscriptionId, location string, callback chan<- func()) {
	subscriptionResourceId := fmt.Sprintf("/subscriptions/%v", subscriptionId)
	complianceClient := security.NewCompliancesClient(subscriptionResourceId, location)
	complianceClient.Authorizer = AzureAuthorizer

	complienceResult, err := complianceClient.Get(context, subscriptionResourceId, time.Now().Format("2006-01-02Z"))
	if err != nil {
		ErrorLogger.Error(fmt.Sprintf("subscription[%v]", subscriptionId), err)
		return
	}

	if complienceResult.AssessmentResult != nil {
		for _, result := range *complienceResult.AssessmentResult {
			segmentType := ""
			if result.SegmentType != nil {
				segmentType = *result.SegmentType
			}

			infoLabels := prometheus.Labels{
				"subscriptionID": subscriptionId,
				"assessmentType": segmentType,
			}
			infoValue := *result.Percentage

			callback <- func() {
				prometheusSecuritycenterCompliance.With(infoLabels).Set(infoValue)
			}
		}
	}
}

func collectAzureAdvisorRecommendations(context context.Context, subscriptionId string, callback chan<- func()) {
	advisorRecommendationsClient := advisor.NewRecommendationsClient(subscriptionId)
	advisorRecommendationsClient.Authorizer = AzureAuthorizer

	recommendationResult, err := advisorRecommendationsClient.ListComplete(context, "", nil, "")
	if err != nil {
		panic(err)
	}

	for _, item := range *recommendationResult.Response().Value {

		infoLabels := prometheus.Labels{
			"subscriptionID": subscriptionId,
			"category":       string(item.RecommendationProperties.Category),
			"resourceType":   *item.RecommendationProperties.ImpactedField,
			"resourceName":   *item.RecommendationProperties.ImpactedValue,
			"resourceGroup":  extractResourceGroupFromAzureId(*item.ID),
			"impact":         string(item.Impact),
			"risk":           string(item.Risk),
		}

		callback <- func() {
			prometheusAdvisorRecommendations.With(infoLabels).Set(1)
		}
	}
}
