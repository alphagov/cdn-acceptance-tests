# -*- mode: ruby -*-
# vi: set ft=ruby :

VAGRANTFILE_API_VERSION = "2"

Vagrant.configure(VAGRANTFILE_API_VERSION) do |config|
  config.vm.box = "ubuntu/trusty64"

  if Vagrant.has_plugin?("vagrant-cachier")
    config.cache.auto_detect = true
  end

  config.vm.network "private_network", ip: "172.16.20.10"

  config.vm.provision "puppet" do |puppet|
    puppet.manifest_file = "site.pp"
    puppet.manifests_path = "mock_cdn_config/manifests"
    puppet.module_path = "mock_cdn_config/modules"
  end
end
