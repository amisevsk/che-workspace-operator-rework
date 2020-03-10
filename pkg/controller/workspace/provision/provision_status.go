package provision

type ProvisioningStatus struct {
	// Continue should be true if cluster state matches spec state for this step
	Continue bool
	Requeue  bool
	Err      error
}

