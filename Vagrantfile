#Vagrant.configure("2") do |config|
#  config.vm.box = "centos/7"
#end
Vagrant.configure("2") do |config|
  config.vm.provider "libvirt" do |v|
    v.memory = 2048
    v.cpus = 2
  end

  config.vm.define "fedora30" do |fedora30|
    fedora30.vm.box = "fedora/30-cloud-base"
  end

  config.vm.define "centos7-pulp-insta-demo" do |a|
    a.vm.box = "centos/7"
    a.vm.provider "libvirt" do |v|
      v.memory = 4096
      v.cpus = 2
    end
    a.vm.provision "shell",
      path: "pulp-insta-demo.sh",
      args: "-f"
  end

  config.vm.define "fedora30-pulp-insta-demo" do |a|
    a.vm.box = "fedora/30-cloud-base"
    a.vm.provider "libvirt" do |v|
      v.memory = 4096
      v.cpus = 2
    end
    a.vm.provision "shell",
      path: "pulp-insta-demo.sh",
      args: "-f"
  end

  config.vm.define "xenial-pulp-insta-demo" do |a|
    a.vm.box = "generic/ubuntu1604"
    a.vm.provider "libvirt" do |v|
      v.memory = 4096
      v.cpus = 2
    end
    a.vm.provision "shell",
      path: "pulp-insta-demo.sh"
  end
end
