#Vagrant.configure("2") do |config|
#  config.vm.box = "centos/7"
#end
Vagrant.configure("2") do |config|
  config.vm.define "fedora30"
  config.vm.box = "fedora/30-cloud-base"
  config.vm.provider "libvirt" do |v|
    v.memory = 2048
    v.cpus = 2
  end
end

Vagrant.configure("2") do |config|
  config.vm.define "fedora30-pulp-insta-demo"
  config.vm.box = "fedora/30-cloud-base"
  config.vm.provider "libvirt" do |v|
    v.memory = 4096
    v.cpus = 2
  end
  config.vm.provision "shell",
    path: "pulp-insta-demo.sh"
end

