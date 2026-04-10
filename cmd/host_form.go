package cmd

type hostFormOptions struct {
	HostNameLabel          string
	HostNameDefault        string
	HostNameRequired       bool
	UserDefault            string
	PortDefault            string
	CurrentAuth            string
	IdentityFileDefault    string
	IdentityFilePromptFrom string
	NormalizeIdentityFile  func(string) (string, error)
	PasswordPrompt         func(auth string) hostFormPasswordResult
}

type hostFormPasswordResult struct {
	PendingPassword string
	Cancelled       bool
	Abort           bool
}

type hostFormData struct {
	HostName        string
	User            string
	Port            string
	Auth            string
	IdentityFile    string
	PendingPassword string
}

func collectHostForm(opts hostFormOptions) (hostFormData, bool, bool, error) {
	hostName, cancelled := promptHostName(opts.HostNameLabel, opts.HostNameDefault, opts.HostNameRequired)
	if cancelled {
		return hostFormData{}, true, false, nil
	}

	user, cancelled := promptUser(opts.UserDefault)
	if cancelled {
		return hostFormData{}, true, false, nil
	}

	port, cancelled := promptPort(opts.PortDefault)
	if cancelled {
		return hostFormData{}, true, false, nil
	}

	auth, cancelled := selectAuthMethod(opts.CurrentAuth)
	if cancelled {
		return hostFormData{}, true, false, nil
	}

	identityFile := opts.IdentityFileDefault
	if auth == "key" {
		identityFile, cancelled = promptIdentityFile(opts.IdentityFilePromptFrom)
		if cancelled {
			return hostFormData{}, true, false, nil
		}
	}

	if opts.NormalizeIdentityFile != nil && identityFile != "" {
		var err error
		identityFile, err = opts.NormalizeIdentityFile(identityFile)
		if err != nil {
			return hostFormData{}, false, false, err
		}
	}

	pendingPassword := ""
	if opts.PasswordPrompt != nil {
		passwordResult := opts.PasswordPrompt(auth)
		if passwordResult.Cancelled {
			return hostFormData{}, true, false, nil
		}
		if passwordResult.Abort {
			return hostFormData{}, false, true, nil
		}
		pendingPassword = passwordResult.PendingPassword
	}

	return hostFormData{
		HostName:        hostName,
		User:            user,
		Port:            port,
		Auth:            auth,
		IdentityFile:    identityFile,
		PendingPassword: pendingPassword,
	}, false, false, nil
}
