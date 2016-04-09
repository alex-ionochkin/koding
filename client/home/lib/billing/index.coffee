kd                                = require 'kd'
HomeTeamBillingPaymentInformation = require './hometeambillingpaymentinformation'
HomeTeamBillingCreditCard         = require './hometeambillingcreditcard'
HomeTeamBillingBillingInformation = require './hometeambillingbillinginformation'

SECTIONS =
  'Payment Information' : HomeTeamBillingPaymentInformation
  'Credit Card'         : HomeTeamBillingCreditCard
  'Billing Information' : HomeTeamBillingBillingInformation


header = (title) ->
  new kd.CustomHTMLView
    tagName  : 'header'
    cssClass : 'HomeAppView--sectionHeader'
    partial  : title


section = (name) -> new SECTIONS[name]


module.exports = class HomeTeamBilling extends kd.CustomScrollView

  constructor: (options = {}, data) ->

    options.cssClass = kd.utils.curry 'HomeAppView--scroller', options.cssClass

    super options, data

    kd.singletons.mainController.ready =>

      formView = new kd.FormView
        cssClass: kd.utils.curry 'HomeAppView--billing-form', options.cssClass

      formView.addSubView header 'Payment Information'
      formView.addSubView section 'Credit Card'
      formView.addSubView section 'Payment Information'

      formView.addSubView header 'Billing Information'
      formView.addSubView section 'Billing Information'

      @wrapper.addSubView formView

