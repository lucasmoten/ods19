Starting with Object Drive 1.0.18, it is built using boringcrypto library for FIPS.
That library is externally prepared, and to bring it in requires use of CGO.
Anytime CGO is used, there are dependencies on GLIBC.

The RPM produced is intended for deployment on Centos6.
Unfortunately, out of the box, Centos6 ships with GLIBC 2.12, and the binary object
for boringcrypto was prepared on a machine running GLIBC 2.14.  As such, when packaged
in the RPM, it requires GLIBC 2.14 or higher as a dependency.

This folder contains GLIBC 2.15 to satisfy this requirement, and is the same
packages referenced in docker/packagingbc used to test this.

The jenkins job will save these artifacts, and the salt environment should install
them as dependencies prior to installing object drive as a service.

rpm -Uvh glibc-2.15-60.el6.x86_64.rpm glibc-common-2.15-60.el6.x86_64.rpm glibc-devel-2.15-60.el6.x86_64.rpm glibc-headers-2.15-60.el6.x86_64.rpm