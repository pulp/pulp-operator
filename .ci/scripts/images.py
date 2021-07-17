import aiohttp
import asyncio
import os
import yaml
from aiohttp.client_exceptions import ClientResponseError
from packaging.version import Version
from packaging.requirements import Requirement

PYPI_ROOT = "https://pypi.org/pypi/{}/json"
PULP_PLUGINS = [
    "pulp-ansible",
    "pulp-certguard",
    "pulp-container",
    "pulp-deb",
    "pulp-file",
    "pulp-python",
    "pulp-rpm",
]
PULP_PLUGINS_WITH_WEBSERVER_SNIPPETS = [
    "pulp_ansible",
    "pulp_container",
    "pulp_python",
]
GALAXY_PLUGINS_WITH_WEBSERVER_SNIPPETS = [
    "galaxy_ng",
    "pulp_ansible",
    "pulp_container",
]


def sort_releases(releases):
    return sorted(releases.keys(), key=lambda ver: Version(ver), reverse=True)


async def get_pypi_data(url):
    async with aiohttp.ClientSession() as session:
        async with session.get(url) as response:
            pypi_data = await response.json()
            return pypi_data


async def get_compatible_plugins(pulpcore_releases, total_releases=1):
    pypi_plugins_data = []
    for plugin in PULP_PLUGINS:
        pkg_url = PYPI_ROOT.format(plugin)
        try:
            pypi_plugins_data.append(get_pypi_data(pkg_url))
        except ClientResponseError as exc:
            if 404 == exc.status:
                print(f"{plugin}  not found on PyPI")
                continue

    done, _ = await asyncio.wait(pypi_plugins_data)
    pypi_plugins_data = [i.result() for i in done]

    for pulpcore_version in pulpcore_releases[0:total_releases]:
        images = []
        for pypi_data in pypi_plugins_data:
            plugin = pypi_data["info"]["name"]
            latest_plugin_version = pypi_data["info"]["version"]
            plugin_versions = sort_releases(pypi_data["releases"])

            for plugin_version in plugin_versions:
                if plugin_version == latest_plugin_version:
                    plugin_requirements = pypi_data["info"]["requires_dist"]
                else:
                    req_data = await get_pypi_data(PYPI_ROOT.format(f"{plugin}/{plugin_version}"))
                    plugin_requirements = req_data["info"]["requires_dist"]
                if "pulpcore-plugin" in str(plugin_requirements):
                    break
                if not plugin_requirements:
                    images.append(f"{plugin}=={plugin_version}")
                    break
                pulpcore_req_for_plugin = Requirement(
                    [r for r in plugin_requirements if "pulpcore" in r][0]
                )
                if Version(pulpcore_version) in pulpcore_req_for_plugin.specifier:
                    images.append(f"{plugin}=={plugin_version}")
                    break

        if len(PULP_PLUGINS) != len(images):
            return

        return images


def save_vars(image, tag, pulpcore, plugins):
    if image == "pulp":
        pulp_vars = yaml.safe_load(open(".ci/ansible/pulp/vars.yaml"))
        pulp_vars["images"].append({
            'pulp_stable': {
                'image_name': 'pulp',
                'tag': tag,
                'container_file': 'Containerfile.core',
                'pulpcore': pulpcore,
                'plugins': plugins,
            }
        })
        pulp_web_vars = yaml.safe_load(open(".ci/ansible/pulp/web/vars.yaml"))
        pulp_web_vars["images"].append({
            'pulp_web_stable': {
                'image_name': 'pulp-web',
                'tag': tag,
                'container_file': 'Containerfile.web',
                'base_image_name': 'pulp',
                'python_version': '3.9',
                'plugin_snippets': PULP_PLUGINS_WITH_WEBSERVER_SNIPPETS,
            }
        })
        yaml.dump(pulp_vars, open(".ci/ansible/pulp/vars.yaml", "w"))
        yaml.dump(pulp_web_vars, open(".ci/ansible/pulp/web/vars.yaml", "w"))

        path = "$GITHUB_WORKSPACE/.ci/scripts/quay-push.sh"
        line = f'sudo -E QUAY_REPO_NAME=pulp QUAY_IMAGE_TAG="{tag}" \{path}'
        os.system(f"echo {line} >> .ci/scripts/deploy.sh")
        line = f'sudo -E QUAY_REPO_NAME=pulp-web QUAY_IMAGE_TAG="{tag}" \{path}'
        os.system(f"echo {line} >> .ci/scripts/deploy.sh")

    if image == "galaxy":
        galaxy_vars = yaml.safe_load(open(".ci/ansible/galaxy/vars.yaml"))
        galaxy_vars["images"].append({
            'galaxy_stable': {
                'image_name': 'galaxy',
                'tag': tag,
                'container_file': 'Containerfile.core',
                'pulpcore': pulpcore,
                'plugins': plugins,
            }
        })
        galaxy_web_vars = yaml.safe_load(open(".ci/ansible/galaxy/web/vars.yaml"))
        galaxy_web_vars["images"].append({
            'galaxy_web_stable': {
                'image_name': 'galaxy-web',
                'tag': tag,
                'container_file': 'Containerfile.web',
                'base_image_name': 'galaxy',
                'python_version': '3.9',
                'plugin_snippets': GALAXY_PLUGINS_WITH_WEBSERVER_SNIPPETS,
            }
        })
        yaml.dump(galaxy_vars, open(".ci/ansible/galaxy/vars.yaml", "w"))
        yaml.dump(galaxy_web_vars, open(".ci/ansible/galaxy/web/vars.yaml", "w"))

        path = "$GITHUB_WORKSPACE/.ci/scripts/quay-push.sh"
        line = f'sudo -E QUAY_REPO_NAME=galaxy QUAY_IMAGE_TAG="{tag}" \{path}'
        os.system(f"echo {line} >> .ci/scripts/deploy.sh")
        line = f'sudo -E QUAY_REPO_NAME=galaxy-web QUAY_IMAGE_TAG="{tag}" \{path}'
        os.system(f"echo {line} >> .ci/scripts/deploy.sh")


if __name__ == "__main__":
    ci_test = os.environ.get("CI_TEST")
    if ci_test == "galaxy":
        galaxy_url = PYPI_ROOT.format("galaxy_ng")
        response = asyncio.run(get_pypi_data(galaxy_url))
        pulpcore_req = [r for r in response["info"]["requires_dist"] if "pulpcore" in r][0]
        pulpcore_req = pulpcore_req.replace(" (", "").replace(")", "")
        save_vars(
            image="galaxy",
            tag=response["info"]["version"],
            pulpcore=f'"{pulpcore_req}"',
            plugins=[f"galaxy_ng=={response['info']['version']}"]
        )
        print(f"galaxy_ng=={response['info']['version']} {pulpcore_req}")
    else:
        pulpcore_url = PYPI_ROOT.format("pulpcore")
        response = asyncio.run(get_pypi_data(pulpcore_url))
        pulpcore_releases = sort_releases(response["releases"])
        plugins = asyncio.run(get_compatible_plugins(pulpcore_releases))
        if plugins:
            save_vars(
                image="pulp",
                tag=response["info"]["version"],
                pulpcore=f"pulpcore=={response['info']['version']}",
                plugins=plugins
            )
            print(f"pulpcore=={response['info']['version']} {plugins}")
