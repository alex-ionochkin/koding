kd                         = require 'kd'
HomeAccountEditProfile     = require './homeaccounteditprofile'
HomeAccountChangePassword  = require './homeaccountchangepassword'
HomeAccountSecurityView    = require './homeaccountsecurityview'
HomeAccountSessionsView    = require './homeaccountsessionsview'
HomeAccountCredentialsView = require './credentials/homeaccountcredentialsview'


SECTIONS =
  Profile     : HomeAccountEditProfile
  Password    : HomeAccountChangePassword
  Security    : HomeAccountSecurityView
  Credentials : HomeAccountCredentialsView
  Sessions    : HomeAccountSessionsView

section = (name) ->
  new (SECTIONS[name] or kd.View)
    tagName  : 'section'
    cssClass : "HomeAppView--section #{kd.utils.slugify name}"


module.exports = class HomeAccount extends kd.CustomScrollView

  constructor: (options = {}, data) ->

    options.cssClass = kd.utils.curry 'HomeAppView--scroller', options.cssClass

    super options, data

    @wrapper.addSubView profile     = section 'Profile'
    @wrapper.addSubView password    = section 'Password'
    @wrapper.addSubView security    = section 'Security'
    @wrapper.addSubView credentials = section 'Credentials'
    @wrapper.addSubView sessions    = section 'Sessions'
