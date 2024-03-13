.PHONY: rpm
rpm:
	$(shell /usr/bin/bash ./update-spec-version.sh)
	spectool -g buildah.spec
	rpmbuild -ba \
		--define '_sourcedir $(shell pwd)' \
		--define '_rpmdir %{_sourcedir}/RPMS' \
		--define '_srcrpmdir %{_sourcedir}/SRPMS' \
		--define '_builddir %{_sourcedir}/BUILD' \
		buildah.spec
	@echo ___RPMS can be found in rpm/RPMS/.___
	@echo ___Undo any changes to Version, Source0 and %autosetup in rpm/buildah.spec before committing.___
