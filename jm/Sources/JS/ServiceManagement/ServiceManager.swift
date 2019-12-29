//
//  ServiceManager.swift
//  JobServer
//
//  Created by Dmytro Naumov on 15.06.18.
//

import Foundation

class ServiceManager {
    private static let shared = ServiceManager()
    private var serviceWaitingList: [Int: [(AIService?, String?) -> Void]]
    public class func sharedInstance() -> ServiceManager {
        return shared
    }

    private var allServices = [Int: SynchronizedArray<AIService>]()
    private var freeJobServices = [Int: SynchronizedArray<AIService>]()
    var asyncJobIds = Set<Int>()
    var syncJobIds = Set<Int>()
    var timer: DispatchSourceTimer

    private init() {
        let queue = DispatchQueue(label: "jobservices.timer")
        self.timer = DispatchSource.makeTimerSource(queue: queue)
        self.timer.schedule(deadline: .now(), repeating: .seconds(15))
        self.serviceWaitingList = [Int: [(AIService?, String?) -> Void]]()
        self.timer.setEventHandler { [weak self] in
            self?.updateServices()
        }
        self.updateServices()
        self.timer.resume()
    }

    private func updateServices() {
        var runningServices = [AIService]()
        guard let discoveryUrl = URL(string: mainConfig.serviceDiscoveryConfig.serviceServerUrl) else {
            Logger.e("Service discovery server URL is not set")
            return
        }

        var request = URLRequest(url: discoveryUrl)
        request.httpMethod = "GET"

        URLSession.shared.dataTask(with: request, completionHandler: { [weak self] (data, response, error) in
            if error != nil {
                Logger.e(error!.localizedDescription)
                return
            }

            guard let resultData = data else {
                Logger.e("No running services returned from the server")
                return
            }

            if let serivces = try? JSONDecoder().decode([AIService].self, from: resultData) {
                runningServices = serivces
            }
                else {
                    Logger.e("Can't decode services data: \(String(describing: String(data: resultData, encoding: .utf8)))")
            }

            self?.updateServiceDictionary(arr: runningServices)
            if !self!.supportedLogged {
                Logger.d("Supported services: ", self?.allServices.keys ?? "[None]")
                self!.supportedLogged = true
            }
        }).resume()
    }
    var supportedLogged = false

    private func updateServiceDictionary(arr: [AIService]) {
        //make dictionaries from all available services
        var receivedServices = [Int: [AIService]]()

        var newAsync = Set<Int>()
        var newSync = Set<Int>()

        for service in arr {
            let serviceId = service.serviceType
            if receivedServices[serviceId] == nil {
                receivedServices[serviceId] = [AIService]()
            }
            receivedServices[serviceId]?.append(service)
            if service.isAsync {
                newAsync.insert(service.serviceType)
            } else {
                newSync.insert(service.serviceType)
            }
        }
        self.asyncJobIds = newAsync
        self.syncJobIds = newSync

        let allReceivedKeys = receivedServices.keys
        let disapearedKeys = self.allServices.keys.filter({ allReceivedKeys.contains($0) == false })
        let newKeys = allReceivedKeys.filter({ self.allServices.keys.contains($0) == false })
        //remove not available anymore services (as dictionaries)
        for key in disapearedKeys {
            self.allServices.removeValue(forKey: key)
            self.freeJobServices.removeValue(forKey: key)
        }

        //create synchronised array fro each new service type
        for key in newKeys {
            self.allServices[key] = SynchronizedArray<AIService>()
        }

        for jobIdKey in receivedServices.keys {
            let thisServiceReceivedArr = receivedServices[jobIdKey]
            //filter out not available anymore service points
            self.allServices[jobIdKey]?.remove(where: { thisServiceReceivedArr!.contains($0) == false })
            //finter and add new service points
            let thisServiceArr = self.allServices[jobIdKey]
            let newServiceArr = thisServiceReceivedArr?.filter({ thisServiceArr!.contains($0) == false })
            for newService in newServiceArr! {
                self.allServices[jobIdKey]?.append(newService)
            }

            //filter out free services arr
            if self.freeJobServices[jobIdKey] != nil {
                self.freeJobServices[jobIdKey]?.remove(where: { thisServiceReceivedArr!.contains($0) == false })
            }
                else {
                    self.freeJobServices[jobIdKey] = SynchronizedArray<AIService>()
            }
            self.freeJobServices[jobIdKey]!.append(self.allServices[jobIdKey]!.filter({ $0.inUse == false && self.freeJobServices[jobIdKey]?.contains($0) == false }))
            self.processWaitingList() //if something was added - process it
        }
    }

    /// Check if service is supported  (i.e. received from service discivery server). For free  service please ask gerNextFreeService()
    ///
    /// - Parameter serviceId: service ID for check
    /// - Returns: true if service is supported
    func supportService(serviceId: Int) -> Bool {
        var result = false
        if let serviceArr = self.allServices[serviceId] {
            result = serviceArr.count > 0
        }
        return result
    }

    /// Run completion closure on with next available AI service
    ///
    /// - Parameters:
    ///   - serviceId: service type
    ///   - completion: closure which will be run when free AI service is appear, or error in the string parameter
    func getNextFreeService(serviceId: Int, completion: @escaping (_ service: AIService?, _ error: String?) -> Void) {
        if self.supportService(serviceId: serviceId) {
            if self.freeJobServices[serviceId] != nil && self.freeJobServices[serviceId]!.count > 0 {
                self.freeJobServices[serviceId]?.remove(at: 0, completion: { freeServiceItem in //free service is available -> return it
                    freeServiceItem.inUse = true
                    completion(freeServiceItem, nil)
                })
            } else {
                if self.serviceWaitingList[serviceId] == nil {
                    self.serviceWaitingList[serviceId] = [(AIService?, String?) -> Void]()
                }
                self.serviceWaitingList[serviceId]?.append(completion)
            }
        } else { //service is not supported
            completion(nil, "Service is not available")
        }
    }


/// Make service available for futher usage
///
/// - Parameter service: service which was used
    func makeServiceFree(service: AIService) {
        DispatchQueue.global().async {
            if self.supportService(serviceId: service.serviceType) && (self.allServices[service.serviceType]!.contains(service)) { //if service is supporterd and servicepoint is available
                if self.serviceWaitingList[service.serviceType] != nil && self.serviceWaitingList[service.serviceType]!.count > 0 {
                    //if there are waiters - just run first of them from queue
                    let waiter = self.serviceWaitingList[service.serviceType]!.remove(at: 0)
                    waiter(service, nil)
                }
                    else { //return service to free services array
                        service.inUse = false
                        self.freeJobServices[service.serviceType]?.append(service)
                        self.processWaitingList() //if something was added - process it
                }
            }
        }
    }

/// Check  if there are available free service and also waiter for this service in  the queue
///
/// - Parameter serviceId: service type
/// - Returns: true if both are available
    private func hasFreeServiceAndWaiter(serviceId: Int) -> Bool {
        let hasFreeService = self.freeJobServices[serviceId] != nil && self.freeJobServices[serviceId]!.count > 0
        let hasWaiter = self.serviceWaitingList[serviceId] != nil && self.serviceWaitingList[serviceId]!.count > 0
        let result = hasFreeService && hasWaiter
        return result
    }

/// Check if there are something in the waiting list and proceed it if workers are available
    private func processWaitingList() {
        for key in self.serviceWaitingList.keys {
            while self.hasFreeServiceAndWaiter(serviceId: key) {
                let waiter = self.serviceWaitingList[key]!.remove(at: 0)

                self.freeJobServices[key]?.remove(at: 0, completion: { freeServiceItem in //free service is available -> return it
                    freeServiceItem.inUse = true
                    waiter(freeServiceItem, nil)
                })
            }
        }
    }
}
