'use strict';

angular.module('chimeraErrorApp', [
		'ngResource',
		'ngSanitize',
        'ngAnimate',
        'ui.router',
        'ui.bootstrap',
	])
	.config(function($stateProvider, $urlRouterProvider, $locationProvider, $httpProvider) {
	    $urlRouterProvider.otherwise('server-error');
	    $locationProvider.html5Mode( false );
    })
    .config(function($stateProvider) {
        $stateProvider.state('access-required', {
            url : '/access-required',
            templateUrl : 'error/accessRequired.html',
            title: 'Access Required'
        }).state('pki-not-found', {
            url : '/pki-not-found',
            templateUrl : 'error/pkiNotFound.html',
            title: 'PKI Not Found'
        }).state('server-error', {
            url : '/server-error',
            templateUrl : 'error/serverError.html',
            title: 'Server Error'
        });
    })
.directive('errorBadge', function() {
        return {
            templateUrl: 'error/errorBadge.html',
            restrict: 'EA',
            scope: {
                icon: '@'
            },
            controller: function($scope, $timeout) {
                $timeout(activate, 2500);
                $scope.activated = false;

                $scope.activate = activate;
                function activate() {
                    $scope.activated = true;
                }
            },
            link: function(scope, element, attrs) {
            }
        };
    });





