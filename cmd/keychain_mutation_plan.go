package cmd

func buildEditKeychainMutations(
	oldAlias string,
	newAlias string,
	oldAuth string,
	newAuth string,
	pendingPassword string,
	existingPassword string,
	hasExistingPassword bool,
) []keychainMutation {
	var mutations []keychainMutation

	if pendingPassword != "" {
		mutations = append(mutations, setPasswordMutation(newAlias, pendingPassword))
	}

	if oldAuth == "password" && newAuth != "password" {
		mutations = append(mutations, deletePasswordMutation(oldAlias))
	}

	if newAlias != oldAlias && newAuth == "password" {
		if pendingPassword == "" && hasExistingPassword {
			mutations = append(mutations, setPasswordMutation(newAlias, existingPassword))
		}
		mutations = append(mutations, deletePasswordMutation(oldAlias))
	}

	return mutations
}

func buildAuthKeychainMutations(alias, oldAuth, newAuth, pendingPassword string) []keychainMutation {
	var mutations []keychainMutation

	if pendingPassword != "" {
		mutations = append(mutations, setPasswordMutation(alias, pendingPassword))
	}

	if oldAuth == "password" && newAuth != "password" {
		mutations = append(mutations, deletePasswordMutation(alias))
	}

	return mutations
}
